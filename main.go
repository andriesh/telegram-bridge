package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type DiscordWebhook struct {
	Content string `json:"content"`
	Embeds  []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"embeds"`
}

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

func main() {
	botToken := os.Getenv("BOT_TOKEN")
	chatID := os.Getenv("CHAT_ID")

	if botToken == "" || chatID == "" {
		log.Fatal("BOT_TOKEN and CHAT_ID must be set")
	}

	http.HandleFunc("/discord", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", 500)
			return
		}

		var payload DiscordWebhook
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid json", 400)
			return
		}

		message := buildMessage(payload)

		if err := sendToTelegram(botToken, chatID, message); err != nil {
			log.Println("telegram send error:", err)
			http.Error(w, "telegram error", 500)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func buildMessage(d DiscordWebhook) string {
	msg := d.Content

	// If embeds exist, prefer them (common for Flagger-style formatting)
	for _, e := range d.Embeds {
		if e.Title != "" {
			msg += "\n\n" + e.Title
		}
		if e.Description != "" {
			msg += "\n" + e.Description
		}
	}

	if msg == "" {
		msg = "Empty Discord webhook payload"
	}

	return msg
}

func sendToTelegram(botToken, chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := TelegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	b, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram error: %s", string(body))
	}

	return nil
}