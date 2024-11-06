package main

import (
    "embed"
    "encoding/json"
    "fmt"
    "log"
    "net"
    "net/http"
    "sync"
    "time"

    "github.com/micmonay/keybd_event"
)

// 静的ファイルを埋め込み
//go:embed web/index.html web/script.js web/styles.css
var staticFiles embed.FS

type KeyRequest struct {
    Key string `json:"key"`
}

// findAvailablePort: 指定されたポートが使用中の場合、別の空いているポートを返す
func findAvailablePort(preferredPort int) int {
    for port := preferredPort; port < 65535; port++ {
        ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
        if err == nil {
            ln.Close()
            return port
        }
    }
    return 0 // 全ポート使用中の場合（ただし現実的には起こりにくい）
}

// Greeting: 起動時の挨拶、WebGUIの案内、自ホストのIPアドレスの表示
func Greeting(port int) {
    log.Printf("サーバーを起動中です... WebGUIは http://0.0.0.0:%d でアクセスできます", port)
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
    file := r.URL.Path[1:]
    if file == "" {
        file = "index.html"
    }
    content, err := staticFiles.ReadFile(file)
    if err != nil {
        http.NotFound(w, r)
        return
    }
    w.Write(content)
}

// キー入力APIサーバー
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

func main() {
    var wg sync.WaitGroup
    wg.Add(2)

    // WebGUIとAPIサーバーのポートを自動調整
    webGUIPort := findAvailablePort(5555)
    apiPort := findAvailablePort(5556)

    // 起動時の挨拶とIPアドレスの表示
    Greeting(webGUIPort)

    // WebGUIサーバー
    go func() {
        defer wg.Done()
        http.HandleFunc("/", webGUIServer)
        log.Printf("WebGUIをポート%dで起動しました", webGUIPort)
        if err := http.ListenAndServe(fmt.Sprintf(":%d", webGUIPort), nil); err != nil {
            log.Fatalf("WebGUIの起動に失敗しました: %v", err)
        }
    }()

    // キー入力APIサーバー
    go func() {
        defer wg.Done()
        http.HandleFunc("/press_key", pressKey)
        http.HandleFunc("/ping", pingHandler)
        log.Printf("キー入力APIサーバーをポート%dで起動しました", apiPort)
        if err := http.ListenAndServe(fmt.Sprintf(":%d", apiPort), nil); err != nil {
            log.Fatalf("キー入力APIサーバーの起動に失敗しました: %v", err)
        }
    }()

    wg.Wait()
}
