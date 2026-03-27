package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type EmailInfo struct {
	Email               string `json:"email"`
	EmailListApi        string `json:"email_list_api"`
	EmailAttachmentsApi string `json:"email_attachment_api"`
}

type GmailMessageList struct {
	Messages          []Message `json:"messages"`
	NextPageToken     string    `json:"nextPageToken"`
	ResultSizeEstimat int       `json:"resultSizeEstimate"`
}

type Message struct {
	Id           string      `json:"id"`
	ThreadId     string      `json:"threadId"`
	Payload      MessagePart `json:"payload"`
	SizeEstimate int         `json:"sizeEstimate"`
	InternalDate int         `json:"internalDate"`
}

type MessagePart struct {
	PartId       string          `json:"partId"`
	Body         MessagePartBody `json:"body"`
	Filename     string          `json:"filename"`
	MessageParts []MessagePart   `json:"parts"`
}

type MessagePartBody struct {
	AttachmentId string `json:"attachmentId"`
	Size         int    `json:"size"`
	Data         string `json:"data"`
}

type tokenSourceWrapper struct {
	context   context.Context
	config    *oauth2.Config
	tknSource oauth2.TokenSource
}

func (wrap *tokenSourceWrapper) Token() (*oauth2.Token, error) {
	tok, err := wrap.tknSource.Token()

	if err != nil {
		return nil, err
	}

	err = saveNewToken(tok)
	if err != nil {
		return nil, err
	}

	return tok, nil
}

func saveNewToken(token *oauth2.Token) error {
	file, err := os.OpenFile("./secret/token.json", os.O_TRUNC|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	err = encoder.Encode(token)
	if err != nil {
		return err
	}
	return nil
}

func getInfoFromJson() (EmailInfo, oauth2.Token) {
	filepathEmail := "./secret/email.json"
	filepathToken := "./secret/token.json"
	emailData, err := os.ReadFile(filepathEmail)
	tokenData, err := os.ReadFile(filepathToken)
	if err != nil {
		panic(err)
	}
	emailJson := EmailInfo{}
	json.Unmarshal(emailData, &emailJson)
	tokenJson := oauth2.Token{}
	json.Unmarshal(tokenData, &tokenJson)
	return emailJson, tokenJson
}

func GetPdfFiles(filter_string string, afterDate string, beforeDate string) error {
	emailJson, tokenJson := getInfoFromJson()

	filepath := "./secret/client_secret.json"
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("Error reading client secret files %v", err)
	}

	ctx := context.Background()
	cfg, err := google.ConfigFromJSON(data, "https://www.googleapis.com/auth/gmail.readonly")
	if err != nil {
		return fmt.Errorf("Error configuring Oauth for requests %v", err)
	}

	tokenSource := cfg.TokenSource(ctx, &tokenJson)
	wrap := tokenSourceWrapper{
		context:   ctx,
		config:    cfg,
		tknSource: tokenSource,
	}
	newClient := oauth2.NewClient(ctx, &wrap)

	// Construct URL
	fullListURL := strings.ReplaceAll(emailJson.EmailListApi, "{userId}", emailJson.Email)
	if afterDate == "" || beforeDate == "" {
		afterDate = "after:" + time.Now().Add(-604800).Format("2006/01/02")
		beforeDate = "before" + time.Now().Add(604800).Format("2006/01/02")
	}

	params := url.Values{}
	params.Add("maxResults", "2")

	// Here goes the email filter
	params.Add("q", filter_string+" "+afterDate+" "+beforeDate)
	params.Add("includeSpamTrash", "false")
	req, err := http.NewRequest("GET", fullListURL+"?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("Error creating request for getting emails that match filter: %v", err)
	}
	res, err := newClient.Do(req)
	if err != nil {
		return fmt.Errorf("Error making request for getting emails that match filter: %v", err)
	}
	if res.StatusCode > 210 {
		return fmt.Errorf("Bad request %d", res.StatusCode)
	}

	// Get info and obtain first message
	mailBytes, err := io.ReadAll(res.Body)
	mailList := GmailMessageList{}
	json.Unmarshal(mailBytes, &mailList)
	res.Body.Close()
	if len(mailList.Messages) <= 1 {
		return fmt.Errorf("Message list is empty.")
	}
	for _, mssg := range mailList.Messages {
		pdfBytes, pdfName, err := lookAtMessageContent(newClient, mssg, fullListURL)
		if err != nil {
			return err
		}
		writePdf(pdfBytes, pdfName)
	}

	return nil
}

func lookAtMessageContent(client *http.Client, mssg Message, fullListURL string) ([]byte, string, error) {
	var pdfBytes []byte
	var pdfName string
	params := url.Values{}
	params.Add("format", "full")
	params.Add("id", mssg.Id)
	req, err := http.NewRequest("GET", fullListURL+"/"+mssg.Id+"?"+params.Encode(), nil)
	if err != nil {
		log.Printf("Problem creating request object: %v \n", err)
		return nil, "", nil
	}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Problem making request: %v \n", err)
		return nil, "", nil
	}
	if res.StatusCode > 210 {
		return pdfBytes, "", fmt.Errorf("Bad request for looking at message content %s", res.Status)
	}

	messageInfo := Message{}
	messageBytes, err := io.ReadAll(res.Body)
	json.Unmarshal(messageBytes, &messageInfo)
	res.Body.Close()
	for _, messagePart := range messageInfo.Payload.MessageParts {

		if !strings.Contains(messagePart.Filename, "pdf") || messagePart.Body.AttachmentId == "" {
			continue
		}
		pdfName = messagePart.Filename
		pdfBytes, err = getEmailAttachmentData(client, fullListURL+"/"+mssg.Id, messagePart.Body.AttachmentId)
		if err != nil {
			return nil, "", fmt.Errorf("Problem getting attachment %s: %v", pdfName, err)
		}
	}
	return pdfBytes, pdfName, nil
}

func getEmailAttachmentData(client *http.Client, url string, attachmentId string) ([]byte, error) {

	var dataUnencodedBytes []byte
	req, err := http.NewRequest("GET", url+"/attachments/"+attachmentId, nil)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	msgPartBodyJson := MessagePartBody{}
	resBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(resBytes, &msgPartBodyJson)
	dataUnencodedBytes, err = base64.URLEncoding.DecodeString(msgPartBodyJson.Data)
	if err != nil {
		return nil, err
	}
	return dataUnencodedBytes, nil
}
