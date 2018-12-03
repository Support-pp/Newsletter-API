package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	rl "github.com/ahmedash95/ratelimit"
	"github.com/gorilla/mux"
)

var ratelimit rl.Limit

type EmailSub struct {
	Status string `json:"status"`
	Email  string `json:"email"`
}

type EmailResponse struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_address"`
	Fields       struct {
		FirstName interface{} `json:"FirstName"`
		LastName  interface{} `json:"LastName"`
	} `json:"fields"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	fmt.Println("[Newsletter] Start Support++ newsletter API. \n Port :: " + os.Getenv("PORT"))
	ratelimit = rl.CreateLimit("1r/s,spam:3,block:14d")
	router := mux.NewRouter()

	router.Handle("/newsletter",
		RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			email := r.Header.Get("email")
			if email == "" {
				w.WriteHeader(400)
				return
			}

			fmt.Println("Create new subscription: " + email)
			sb := addEMailToList(email)
			out, _ := json.Marshal(&sb)
			w.Write([]byte(out))
		}))).Methods("POST")

	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}

func addEMailToList(email string) EmailSub {

	url := "https://emailoctopus.com/api/1.5/lists/d45abf77-f709-11e8-a3c9-06b79b628af2/contacts"

	payload := strings.NewReader("api_key=" + os.Getenv("APIKEY") + "&email_address=" + email)

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("cache-control", "no-cache")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	fmt.Println("Request to API! Status :: ")

	data := EmailResponse{}

	json.Unmarshal([]byte(body), &data)
	return EmailSub{
		Email:  data.EmailAddress,
		Status: data.Status,
	}

}

// Middleware
func RateLimitMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := "127.0.0.1" // use ip or user agent any key you want
		if !isValidRequest(ratelimit, ip) {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		ratelimit.Hit(ip)
		h.ServeHTTP(w, r)
	})
}

func isValidRequest(l rl.Limit, key string) bool {
	_, ok := l.Rates[key]
	if !ok {
		return true
	}
	if l.Rates[key].Hits == l.MaxRequests {
		return false
	}
	return true
}
