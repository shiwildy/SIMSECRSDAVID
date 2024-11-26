package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

var conn *websocket.Conn

func main() {
	// Monitor folder RS JAKPUS
	go monitorFolder("rs-jakpus", "10.10.10.1:8443")

	// Setup HTTP server with HSTS and WebSocket upgrade
	http.HandleFunc("/", hstsHandler)
	http.HandleFunc("/ws", handleWebSocket)

	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: configureTLS("certs/secret-rsjakut.crt", "certs/secret-rsjakut.key", "certs/secret-pubrsjakpus.crt"),
	}

	log.Println("Server RS JAKUT listening on :8443 with TLS 1.3 and HSTS")
	log.Fatal(server.ListenAndServeTLS("certs/secret-rsjakut.crt", "certs/secret-rsjakut.key"))
}

func hstsHandler(w http.ResponseWriter, r *http.Request) {
	// Tambahkan header HSTS
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	fmt.Fprintf(w, "RS JAKUT HTTP Server with HSTS enabled")
}

func monitorFolder(folderPath, remoteAddr string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Error initializing watcher:", err)
	}
	defer watcher.Close()

	err = watcher.Add(folderPath)
	if err != nil {
		log.Fatal("Error adding folder to watcher:", err)
	}

	// Hubungkan ke server RS JAKPUS dengan TLS
	conn = connectToServer(remoteAddr)
	defer conn.Close()

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("Detected new file:", event.Name)
				sendFile(event.Name)
			}
		case err := <-watcher.Errors:
			log.Println("Error watching folder:", err)
		}
	}
}

func connectToServer(remoteAddr string) *websocket.Conn {
	caCert, err := ioutil.ReadFile("certs/secret-pubrsjakpus.crt")
	if err != nil {
		log.Fatal("Error loading CA certificate:", err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair("certs/secret-rsjakut.crt", "certs/secret-rsjakut.key")
	if err != nil {
		log.Fatal("Error loading key pair:", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
	}

	conn, _, err := dialer.Dial("wss://"+remoteAddr+"/ws", nil)
	if err != nil {
		log.Fatal("Error connecting to RS JAKPUS:", err)
	}
	return conn
}

func configureTLS(certFile, keyFile, caCertFile string) *tls.Config {
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		log.Fatal("Error loading CA certificate:", err)
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal("Error loading key pair:", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}
}

func sendFile(filePath string) {
	// Validasi file sebelum dikirim
	fileInfo, err := os.Stat(filePath)
	if err != nil || fileInfo.IsDir() {
		log.Println("Error: Invalid file or directory:", filePath)
		return
	}

	// Kirim nama file
	err = conn.WriteMessage(websocket.TextMessage, []byte(filepath.Base(filePath)))
	if err != nil {
		log.Println("Error sending file name:", err)
		return
	}

	// Kirim isi file
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Baca isi file dan kirim melalui WebSocket
	buffer := make([]byte, 4096)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Error reading file:", err)
			return
		}
		err = conn.WriteMessage(websocket.BinaryMessage, buffer[:n])
		if err != nil {
			log.Println("Error sending file data:", err)
			return
		}
	}
	log.Println("File sent successfully:", filePath)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading WebSocket:", err)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
		log.Println("Received message:", string(msg))
	}
}
