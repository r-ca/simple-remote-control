package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/micmonay/keybd_event"
)

// ANSIカラーコード
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

const (
	VERSION         = "0.2.3"
	DEFAULT_PORT    = 8080
	DEFAULT_ADDRESS = "0.0.0.0"
)

// Ping log flag
var pingLog = false

// 静的ファイルを埋め込み
//
//go:embed web/*
var staticFiles embed.FS

type KeyRequest struct {
	Key string `json:"key"`
}

// カラーログ関数
func logInfo(message string) {
	log.Println(colorGreen + message + colorReset)
}

func logWarn(message string) {
	log.Println(colorYellow + message + colorReset)
}

func logError(message string) {
	log.Println(colorRed + message + colorReset)
}

// WebGUI（静的ファイルサーバー）
func webGUIServer(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Path
	if file == "/" {
		file = "/index.html"
	}

	// MIMEタイプの設定
	ext := filepath.Ext(file)
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}

	// webディレクトリを基準にファイルパスを構築
	content, err := staticFiles.ReadFile(path.Join("web", file))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Write(content)
}

// キー入力API
func pressKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req KeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "リクエストが不正です", http.StatusBadRequest)
		logWarn("リクエストが不正です: " + err.Error())
		return
	}

	if req.Key != "" {
		err := sendKeyEvent(req.Key)
		if err != nil {
			http.Error(w, "キー入力に失敗しました", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("キーが入力されました: " + req.Key))
		logInfo("キーが入力されました: " + req.Key + " (Source: " + r.RemoteAddr + ")")
	} else {
		http.Error(w, "キーが指定されていません", http.StatusBadRequest)
	}
}

// キー入力操作
func sendKeyEvent(key string) error {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		return err
	}

	switch key {
	case "left":
		kb.SetKeys(keybd_event.VK_LEFT)
	case "right":
		kb.SetKeys(keybd_event.VK_RIGHT)
	default:
		return nil
	}

	err = kb.Launching()
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond)
	return nil
}

// Pingエンドポイントのハンドラー
func pingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Write([]byte("pong"))
	if pingLog {
		logInfo("Pingを受信しました (Source: " + r.RemoteAddr + ")")
	}
}

// サーバーを開始する関数
func startServer(address string, port int) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webGUIServer)
	mux.HandleFunc("/api/press_key", pressKey)
	mux.HandleFunc("/api/ping", pingHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", address, port),
		Handler: mux,
	}

	go func() {
		logInfo(fmt.Sprintf("サーバーを %s で起動しました", server.Addr))
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			logError(fmt.Sprintf("サーバーの起動に失敗しました: %v", err))
		}
	}()

	return server
}

// コマンドからインタフェースのIPアドレスを表示
func showLocalIPs() {
	ifaces, err := net.Interfaces()
	if err != nil {
		logError(fmt.Sprintf("IPアドレスの取得に失敗しました: %v", err))
		return
	}

	logInfo("自ホストのIPアドレス一覧:")
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			logError(fmt.Sprintf("インタフェース %v のアドレス取得に失敗しました: %v", iface.Name, err))
			continue
		}
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				logError(fmt.Sprintf("アドレスの解析に失敗しました: %v", err))
				continue
			}
			if ip.IsLoopback() {
				continue
			}
			fmt.Printf("- %s: %s\n", iface.Name, ip.String())
		}
	}
}

// ターミナルからのコマンド入力を処理する関数
func commandLoop(cancel context.CancelFunc, server **http.Server, address *string, port *int) {
	scanner := bufio.NewScanner(os.Stdin)
	// fmt.Println("コマンドを入力してください（'show'でインタフェースのアドレスを表示、'switch <port>'でポート切り替え、'exit'で終了、'log ping <on|off>'でPingログモードの制御）：")

	fmt.Println("コマンドを入力... (help: ヘルプ)")

	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		parts := strings.Split(input, " ")

		switch parts[0] {
		case "show":
			showLocalIPs()
		case "switch":
			if len(parts) == 2 {
				newPort, err := strconv.Atoi(parts[1])
				if err == nil {
					logWarn(fmt.Sprintf("サーバーをポート %d で再起動します...", newPort))
					(*server).Shutdown(context.Background()) // 現在のサーバーをシャットダウン
					*port = newPort
					*server = startServer(*address, *port) // 新しいポートでサーバーを再起動
					logInfo(fmt.Sprintf("新しいポート %d でサーバーが起動しました", newPort))
				} else {
					fmt.Println("無効なポート番号です。正しい整数を入力してください。")
				}
			} else {
				fmt.Println("使用法: switch <port>")
			}
		case "exit":
			fmt.Println("終了します。")
			cancel() // サーバーのシャットダウンをトリガー
			return
		case "log":
			if len(parts) == 3 {
				switch parts[1] {
				case "ping":
					if parts[2] == "on" {
						pingLog = true
						logInfo("Pingログを有効にしました")
					} else if parts[2] == "off" {
						pingLog = false
						logInfo("Pingログを無効にしました")
					} else {
						fmt.Println("使用法: log ping <on|off>")
					}
				case "key":
					// Not implemented
					logWarn("WIP")
				default:
					fmt.Println("使用法: log <ping|key> <on|off>")
				}
			} else {
				fmt.Println("使用法: log <ping|key> <on|off>")
			}

		case "help":
			fmt.Println("・ 'show'              : ホストに搭載されているI/Fのアドレスを列挙")
			fmt.Println("・ 'switch <port>'     : 使用ポートの切り替え")
			fmt.Println("・ 'log ping <on|off>' : Pingログの表示を切り替え")
			fmt.Println("・ 'exit'              : システムを終了")
		default:
			fmt.Println("不明なコマンドです。(help: ヘルプ)")
		}
		fmt.Println("コマンドを入力... (help: ヘルプ)")
	}

	if err := scanner.Err(); err != nil {
		logError(fmt.Sprintf("コマンド入力中にエラーが発生しました: %v", err))
	}
}

func greeting(address string, port int) {
	fmt.Println("--------------------------------------------------------")
	fmt.Println(" Simple Remote Control v" + VERSION)
	fmt.Println("       シンプルなシングルバイナリ型PowerPointリモコン")
	fmt.Println("--------------------------------------------------------")
	fmt.Printf("- Listen for: %s:%d\n", address, port)
	fmt.Printf("(ヒント: このホストからWebGUIに接続する場合、\n")
	fmt.Printf("  一般的な環境では http://localhost:%d が使用できます\n", port)
	fmt.Println("--------------------------------------------------------")
}

func main() {
	address := flag.String("addr", DEFAULT_ADDRESS, "サーバーのアドレス")
	port := flag.Int("port", DEFAULT_PORT, "サーバーのポート番号")
	flag.Parse()

	greeting(*address, *port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	server := startServer(*address, *port)

	go commandLoop(cancel, &server, address, port)

	select {
	case <-sigs:
		fmt.Println("\nシグナルを受信しました。終了します。")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logError(fmt.Sprintf("サーバーのシャットダウンに失敗しました: %v", err))
	}
	logInfo("サーバーが正常に停止しました。")

	// Bye! log
	logInfo("Bye!")
}
