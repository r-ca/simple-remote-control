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
    "strings"
    "syscall"
    "time"

    "github.com/micmonay/keybd_event"
)

// 静的ファイルを埋め込み
//go:embed web/*
var staticFiles embed.FS

type KeyRequest struct {
    Key string `json:"key"`
}

// Greeting: 起動時の挨拶とIPアドレスの表示
func Greeting(address string, port int) {
    log.Printf("サーバーを起動中です... WebGUIは http://%s:%d でアクセスできます", address, port)
    listLocalIPs()
}

// 自ホストのIPアドレスを列挙
func listLocalIPs() {
    ifaces, err := net.Interfaces()
    if err != nil {
        log.Printf("IPアドレスの取得に失敗しました: %v\n", err)
        return
    }

    log.Println("自ホストのIPアドレス一覧:")
    for _, iface := range ifaces {
        addrs, err := iface.Addrs()
        if err != nil {
            log.Printf("インタフェース %v のアドレス取得に失敗しました: %v\n", iface.Name, err)
            continue
        }
        for _, addr := range addrs {
            ip, _, err := net.ParseCIDR(addr.String())
            if err != nil {
                log.Printf("アドレスの解析に失敗しました: %v\n", err)
                continue
            }
            if ip.IsLoopback() {
                continue
            }
            fmt.Printf("- %s: %s\n", iface.Name, ip.String())
        }
    }
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
    case "a":
        kb.SetKeys(keybd_event.VK_A)
    case "b":
        kb.SetKeys(keybd_event.VK_B)
    case "space":
        kb.SetKeys(keybd_event.VK_SPACE)
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
    w.Write([]byte("pong"))
}

// ターミナルからのコマンド入力を処理する関数
func commandLoop(cancel context.CancelFunc) {
    scanner := bufio.NewScanner(os.Stdin)
    fmt.Println("コマンドを入力してください（'show'でインタフェースのアドレスを表示、'exit'で終了）：")
    for scanner.Scan() {
        input := strings.TrimSpace(scanner.Text())
        switch input {
        case "show":
            listLocalIPs()
        case "exit":
            fmt.Println("終了します。")
            cancel() // サーバーのシャットダウンをトリガー
            return
        default:
            fmt.Println("不明なコマンドです。'show' または 'exit' を入力してください。")
        }
        fmt.Println("コマンドを入力してください（'show'でインタフェースのアドレスを表示、'exit'で終了）：")
    }
    if err := scanner.Err(); err != nil {
        log.Printf("コマンド入力中にエラーが発生しました: %v", err)
    }
}

func main() {
    // コマンドライン引数でアドレスとポートを指定
    address := flag.String("addr", "localhost", "サーバーのアドレス")
    port := flag.Int("port", 5555, "サーバーのポート番号")
    flag.Parse()

    // シグナル処理のためのコンテキスト
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // シグナルのリスニングをセットアップ
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

    // マルチプレクサの設定
    mux := http.NewServeMux()
    mux.HandleFunc("/", webGUIServer)        // WebGUI（静的ファイルサーバー）
    mux.HandleFunc("/api/press_key", pressKey) // キー入力API
    mux.HandleFunc("/api/ping", pingHandler)    // Pingエンドポイント

    // サーバー設定
    server := &http.Server{
        Addr:    fmt.Sprintf("%s:%d", *address, *port),
        Handler: mux,
    }

    // サーバーを別ゴルーチンで起動
    go func() {
        log.Printf("サーバーを %s で起動しました", server.Addr)
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("サーバーの起動に失敗しました: %v", err)
        }
    }()

    // コマンド入力の待機を別ゴルーチンで実行
    go commandLoop(cancel)

    // シグナルまたはキャンセルされるまで待機
    select {
    case <-sigs:
        fmt.Println("\nシグナルを受信しました。終了します。")
    case <-ctx.Done():
    }

    // サーバーのシャットダウン
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer shutdownCancel()
    if err := server.Shutdown(shutdownCtx); err != nil {
        log.Fatalf("サーバーのシャットダウンに失敗しました: %v", err)
    }
    log.Println("サーバーが正常に停止しました。")
}
