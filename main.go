package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/gin-gonic/gin"
	"github.com/go-mail/mail/v2"
	"google.golang.org/api/option"
)

type Config struct {
	Email struct {
		SMTPHost string `json:"smtp_host"`
		SMTPPort int    `json:"smtp_port"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"email"`
	Firebase struct {
		CredentialsFile string `json:"credentials_file"`
	} `json:"firebase"`
}

var config Config

func loadConfig() error {
	file, err := os.Open("config.json")
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	return err
}

func main() {
	if err := loadConfig(); err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	r := gin.Default()

	r.POST("/send-email", SendEmail)
	r.POST("/send-push", SendPushNotification)

	r.Run(":8080")
}

type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type PushNotificationRequest struct {
	Token string `json:"token"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

func SendEmail(c *gin.Context) {
	var req EmailRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	m := mail.NewMessage()
	m.SetHeader("From", config.Email.Username)
	m.SetHeader("To", req.To)
	m.SetHeader("Subject", req.Subject)
	m.SetBody("text/plain", req.Body)

	d := mail.NewDialer(config.Email.SMTPHost, config.Email.SMTPPort,
		config.Email.Username, config.Email.Password)

	if err := d.DialAndSend(m); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email sent successfully"})
}

func SendPushNotification(c *gin.Context) {
	var req PushNotificationRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx := context.Background()
	opt := option.WithCredentialsFile(config.Firebase.CredentialsFile)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize Firebase app"})
		return
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize Firebase messaging client"})
		return
	}

	message := &messaging.Message{
		Token: req.Token,
		Notification: &messaging.Notification{
			Title: req.Title,
			Body:  req.Body,
		},
	}

	_, err = client.Send(ctx, message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send push notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Push notification sent successfully"})
}
