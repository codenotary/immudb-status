package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"flag"
	"github.com/codenotary/immudb/pkg/api/schema"
	immuclient "github.com/codenotary/immudb/pkg/client"
	"google.golang.org/grpc/metadata"
	"log"
	"os"
	"regexp"
)

var (
	Version   = "0.0"
	Buildtime = "00"
	Commit    = "00"
	AESKey    = "NOKEY"
)

var config struct {
	Addr     string
	Port     int
	Username string
	Password string
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	config.Addr = "127.0.0.1"
	if envAddr, ok := os.LookupEnv("IMMUDB_ADDRESS"); ok {
		config.Addr = envAddr
	}
	config.Password = "immudb"
	if envPw, ok := os.LookupEnv("IMMUDB_PASSWORD"); ok {
		config.Password = envPw
	}
	flag.StringVar(&config.Addr, "addr", config.Addr, "IP address of immudb server [IMMUDB_ADDRESS]")
	flag.IntVar(&config.Port, "port", 3322, "Port number of immudb")
	flag.StringVar(&config.Username, "user", "immudb", "Admin username for immudb")
	flag.StringVar(&config.Password, "pass", config.Password, "Admin password for immudb [IMMUDB_PASSWORD]")

	flag.Parse()

	if s, err := aesdecrypt(config.Password); err == nil {
		config.Password = s
	}
}

func aesdecrypt(s string) (string, error) {
	rx := regexp.MustCompile(`^(.*?)enc:([[:xdigit:]]+)(.*)$`)
	m := rx.FindStringSubmatch(s)
	if len(m) == 0 {
		return s, nil
	}
	bs, err := hex.DecodeString(m[2])
	if err != nil {
		return s, err
	}
	block, err := aes.NewCipher([]byte(AESKey))
	if err != nil {
		return s, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return s, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := bs[:nonceSize], bs[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return s, err
	}
	return string(plaintext), nil
}

func connect(addr string, port int, username, password string) (context.Context, immuclient.ImmuClient) {
	ctx := context.Background()
	opts := immuclient.DefaultOptions().WithAddress(addr).WithPort(port)

	client, err := immuclient.NewImmuClient(opts)
	if err != nil {
		log.Printf("Failed to connect to %s:%d. Reason: %s", addr, port, err.Error())
		return ctx, nil
	}

	login, err := client.Login(ctx, []byte(username), []byte(password))
	if err != nil {
		log.Printf("Failed to login to %s:%d. Reason: %s", addr, port, err.Error())
		return ctx, nil
	}
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", login.GetToken()))
	return ctx, client
}

func db_list(ctx context.Context, client immuclient.ImmuClient) []string {
	var databases []string
	dbs, err := client.DatabaseList(ctx)
	if err != nil {
		log.Printf("Failed to get database list. Reason: %s", err.Error())
	}
	for _, s := range dbs.Databases {
		databases = append(databases, s.DatabaseName)
	}
	return databases
}

func main() {
	log.Printf("Immustat %s [%s] @ %s", Version, Commit, Buildtime)
	m_ctx, cli := connect(config.Addr, config.Port, config.Username, config.Password)
	if cli == nil {
		return
	}
	dblist := db_list(m_ctx, cli)
	for _, db := range dblist {
		udr, err := cli.UseDatabase(m_ctx, &schema.Database{DatabaseName: db})
		if err != nil {
			log.Printf("Failed to use the database. Reason: %s", err.Error())
			return
		}
		ctx := metadata.NewOutgoingContext(m_ctx, metadata.Pairs("authorization", udr.GetToken()))
		ret, err := cli.CurrentState(ctx)
		if err != nil {
			log.Printf("Failed to get database status. Reason: %s", err.Error())
			return
		}

		log.Printf("DB: %s TxId :%d, Hash: %s", ret.Db, ret.TxId, hex.EncodeToString(ret.TxHash))
	}
}
