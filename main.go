package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

//
// -------------------- Flagger / Slack payload --------------------
//

type SlackPayload struct {
	Channel     string `json:"channel"`
	Username    string `json:"username"`
	IconEmoji   string `json:"icon_emoji"`
	IconURL     string `json:"icon_url"`
	Text        string `json:"text"`
	Attachments []struct {
		Color      string `json:"color"`
		AuthorName string `json:"author_name"`
		Text       string `json:"text"`
		Fields     []struct {
			Title string `json:"title"`
			Value string `json:"value"`
			Short bool   `json:"short"`
		} `json:"fields"`
	} `json:"attachments"`
}

//
// -------------------- Telegram payload --------------------
//

type TelegramMessage struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

//
// -------------------- main --------------------
//

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	chatID := os.Getenv("CHAT_ID")

	if botToken == "" || chatID == "" {
		log.Fatal("BOT_TOKEN and CHAT_ID are required")
	}

	mux := http.NewServeMux()

	// Flagger actual endpoint (IMPORTANT)
	mux.HandleFunc("/discord/slack", handler(botToken, chatID))

	// manual testing endpoints
	mux.HandleFunc("/discord/webhook", handler(botToken, chatID))
	mux.HandleFunc("/discord", handler(botToken, chatID))

	// health
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Println("listening on :8080")
	log.Println("routes:")
	log.Println("  POST /discord/slack   (Flagger)")
	log.Println("  POST /discord/webhook (test)")
	log.Println("  POST /discord         (fallback)")
	log.Println("  GET  /healthz")

	log.Fatal(http.ListenAndServe(":8080", mux))
}

//
// -------------------- handler --------------------
//

func handler(botToken, chatID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusInternalServerError)
			return
		}

		logIncoming(r, body)

		// parse Slack payload (Flagger format)
		var payload SlackPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf(`{"level":"error","msg":"invalid payload","err":"%s"}`, err)
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		msg := formatSlackToTelegram(payload)

		if err := sendTelegram(botToken, chatID, msg); err != nil {
			log.Printf(`{"level":"error","msg":"telegram send failed","err":"%s"}`, err)
			http.Error(w, "telegram error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}

//
// -------------------- formatting --------------------
//

func formatSlackToTelegram(p SlackPayload) string {
	var b strings.Builder

	if p.Username != "" {
		b.WriteString("🚀 ")
		b.WriteString(p.Username)
		b.WriteString("\n\n")
	}

	if p.Text != "" {
		b.WriteString(p.Text)
		b.WriteString("\n")
	}

	for _, a := range p.Attachments {

		if a.AuthorName != "" {
			b.WriteString("\n📦 ")
			b.WriteString(a.AuthorName)
			b.WriteString("\n")
		}

		if a.Text != "" {
			b.WriteString(a.Text)
			b.WriteString("\n")
		}

		for _, f := range a.Fields {
			b.WriteString("\n")
			b.WriteString("• ")
			b.WriteString(f.Title)
			b.WriteString(": ")
			b.WriteString(f.Value)
		}
	}

	out := strings.TrimSpace(b.String())

	if out == "" {
		return "Empty Flagger notification"
	}

	return out
}

//
// -------------------- telegram --------------------
//

func sendTelegram(botToken, chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := TelegramMessage{
		ChatID: chatID,
		Text:   text,
	}

	b, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

//
// -------------------- logging --------------------
//

func logIncoming(r *http.Request, body []byte) {
	var parsed interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		parsed = string(body)
	}

	entry := map[string]interface{}{
		"level":   "info",
		"path":    r.URL.Path,
		"method":  r.Method,
		"headers": r.Header,
		"body":    parsed,
	}

	b, _ := json.Marshal(entry)
	log.Println(string(b))
}