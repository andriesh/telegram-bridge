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

type DiscordWebhook struct {
	Content string         `json:"content"`
	Embeds  []DiscordEmbed `json:"embeds"`
}

type DiscordEmbed struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       int    `json:"color,omitempty"`
}

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	chatID := os.Getenv("CHAT_ID")

	if botToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	if chatID == "" {
		log.Fatal("CHAT_ID is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/healthz", healthHandler)

	http.HandleFunc("/discord/webhook", func(w http.ResponseWriter, r *http.Request) {
		discordWebhookHandler(w, r, botToken, chatID)
	})

	http.HandleFunc("/", catchAllHandler)

	log.Printf(
		`{"level":"info","message":"server started","port":"%s","endpoint":"/discord/webhook"}`,
		port,
	)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func catchAllHandler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	logRequest(r, body)

	http.NotFound(w, r)
}

func discordWebhookHandler(
	w http.ResponseWriter,
	r *http.Request,
	botToken string,
	chatID string,
) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request", http.StatusInternalServerError)
		return
	}

	logRequest(r, body)

	var payload DiscordWebhook

	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf(
			`{"level":"error","message":"invalid discord payload","error":"%s"}`,
			err.Error(),
		)

		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	message := buildTelegramMessage(payload)

	if err := sendTelegram(botToken, chatID, message); err != nil {
		log.Printf(
			`{"level":"error","message":"telegram send failed","error":"%s"}`,
			err.Error(),
		)

		http.Error(w, "telegram send failed", http.StatusInternalServerError)
		return
	}

	log.Printf(
		`{"level":"info","message":"telegram notification sent"}`,
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func buildTelegramMessage(payload DiscordWebhook) string {
	var b strings.Builder

	if payload.Content != "" {
		b.WriteString(payload.Content)
		b.WriteString("\n")
	}

	for _, embed := range payload.Embeds {
		if embed.Title != "" {
			b.WriteString("\n")
			b.WriteString("📢 ")
			b.WriteString(embed.Title)
			b.WriteString("\n")
		}

		if embed.Description != "" {
			b.WriteString(embed.Description)
			b.WriteString("\n")
		}
	}

	msg := strings.TrimSpace(b.String())

	if msg == "" {
		msg = "Received empty Discord webhook payload"
	}

	return msg
}

func sendTelegram(
	botToken string,
	chatID string,
	text string,
) error {

	url := fmt.Sprintf(
		"https://api.telegram.org/bot%s/sendMessage",
		botToken,
	)

	reqBody := TelegramMessage{
		ChatID: chatID,
		Text:   text,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf(
			"telegram returned %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	return nil
}

func logRequest(r *http.Request, body []byte) {
	headers := map[string]string{}

	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	var bodyJSON interface{}

	if err := json.Unmarshal(body, &bodyJSON); err != nil {
		bodyJSON = string(body)
	}

	entry := map[string]interface{}{
		"level":   "info",
		"message": "incoming request",
		"method":  r.Method,
		"path":    r.URL.Path,
		"headers": headers,
		"body":    bodyJSON,
	}

	b, err := json.Marshal(entry)
	if err != nil {
		log.Printf(
			`{"level":"error","message":"failed to serialize request log","error":"%s"}`,
			err.Error(),
		)
		return
	}

	log.Printf("%s", string(b))
}