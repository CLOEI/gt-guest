package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	api2captcha "github.com/2captcha/2captcha-go"
)

const USERAGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"

func main() {
	var (
		captcha_api_key = os.Getenv("2captcha_api_key")
		should_log      = os.Getenv("should_log")
		f               *os.File
		err             error
	)

	if should_log == "true" {
		f, err = os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("error opening file: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	log.Println("Starting server...")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Has("url") {
			data := query.Get("url")
			fmt.Printf("Working on url: %s\n", data)
			platformid := query.Get("platformID")
			initial_token := parse_url_token(data)
			if initial_token != "" {
				sitekey, _token, token, err := fetch_data(initial_token, platformid)
				if err != nil {
					log.Println(err)
				}
				captcha := solve_captcha(fmt.Sprintf("https://www.growtopiagame.com/player/login?token=%s&platformID=0r", initial_token), sitekey, captcha_api_key)
				validated := validate_captcha(_token, token, captcha, initial_token, platformid)
				if validated {
					w.Write([]byte("Captcha validated!"))
				} else {
					w.Write([]byte("Captcha failed to validate!"))
				}
			}
		} else {
			w.Write([]byte("Add your url as a query parameter. Example: http://localhost:3000?url=https://www.google.com"))
		}
	})

	fmt.Println("2captcha_api_key: ", captcha_api_key)
	fmt.Println("should_log: ", should_log)

	fmt.Println("Server started on port 5000")
	log.Fatal(http.ListenAndServe(":5000", nil))
}

func validate_captcha(_token string, token string, captcha_data string, initial_token string, platformid string) bool {
	log.Println("Validating captcha...")
	fmt.Println("Validating captcha...")
	payload := url.Values{}
	payload.Set("_token", _token)
	payload.Set("g-recaptcha-response", captcha_data)
	payload.Set("token", token)

	req, err := http.NewRequest("POST", "https://www.growtopiagame.com/player/validate", strings.NewReader(payload.Encode()))
	if err != nil {
		log.Println(err)
		return false
	}

	req.Header.Set("User-Agent", USERAGENT)
	req.Header.Set("Host", "www.growtopiagame.com")
	req.Header.Set("Origin", "https://www.growtopiagame.com")
	req.Header.Set("Referer", fmt.Sprintf("https://www.growtopiagame.com/player/login?token=%s&platformID=%s", initial_token, platformid))
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("sec-ch-ua-platform", "Windows")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return false
	}

	fmt.Println(string(body))
	defer resp.Body.Close()
	return true
}

func solve_captcha(url string, sitekey string, api_key string) string {
	client := api2captcha.NewClient(api_key)
	client.RecaptchaTimeout = 500
	cap := api2captcha.ReCaptcha{
		SiteKey:   sitekey,
		Url:       url,
		Invisible: true,
		Action:    "verify",
	}
	req := cap.ToRequest()
	log.Println("Solving captcha...")
	fmt.Println("Solving captcha...")
	token, _, err := client.Solve(req)
	if err != nil {
		log.Println(err)
		return ""
	}
	log.Printf("Captcha token: %s\n", token)
	fmt.Printf("Captcha token: %s\n", token)
	return token
}

func parse_url_token(data string) string {
	parts := strings.Split(data, "?token=")
	if len(parts) > 1 {
		return url.QueryEscape(parts[1])
	}
	return ""
}

func fetch_data(init_token string, plaformid string) (string, string, string, error) {
	log.Println("Fetching data...")
	req, err := http.NewRequest("GET", fmt.Sprintf("https://www.growtopiagame.com/player/login?token=%s&platformID=%s", init_token, plaformid), nil)
	if err != nil {
		log.Println(err)
		return "", "", "", err
	}

	req.Header.Set("User-Agent", USERAGENT)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return "", "", "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return "", "", "", err
	}

	body_string := string(body)
	sitekey_re := regexp.MustCompile(`data-sitekey="([^"]+)`)
	_token_re := regexp.MustCompile(`name="_token" type="hidden" value="([^"]*)"`)
	token_re := regexp.MustCompile(`name="token" type="hidden" value="([^"]*)"`)
	sitekey := sitekey_re.FindStringSubmatch(body_string)
	_token := _token_re.FindStringSubmatch(body_string)
	token := token_re.FindStringSubmatch(body_string)

	log.Printf("sitekey: %s | _token: %s | token: %s\n", sitekey[1], _token[1], token[1])

	defer resp.Body.Close()
	return sitekey[1], _token[1], token[1], nil
}
