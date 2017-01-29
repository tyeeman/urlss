package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goware/urlx"
	"github.com/schollz/jsonstore"
)

var ks *jsonstore.JSONStore

func init() {
	var err error
	ks, err = jsonstore.Open("urls.json.gz")
	if err != nil {
		ks = new(jsonstore.JSONStore)
	}
}

func newShortenedURL() string {
	for n := 2; n < 10; n++ {
		for i := 0; i < 10; i++ {
			candidate := RandString(n)
			var foo string
			err := ks.Get(candidate, &foo)
			if err != nil {
				return candidate
			}
		}
	}
	return ""
}

var Host, Port string

func main() {
	gin.SetMode(gin.ReleaseMode)
	flag.StringVar(&Host, "h", "", "host (optional)")
	flag.StringVar(&Port, "p", "8006", "port (default 8006)")
	flag.Parse()
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/*action", func(c *gin.Context) {
		action := c.Param("action")
		action = action[1:len(action)]
		if strings.Contains(action, "http") && !strings.Contains(action, "//") {
			action = strings.Replace(action, "/", "//", 1)
		}
		parsedURL, _ := urlx.Parse(action)
		url, _ := urlx.Normalize(parsedURL)
		if len(url) > 0 && !strings.Contains(url, "favicon.ico") {
			// Save the URL
			var shortened string
			err := ks.Get(url, &shortened)
			if err == nil {
				c.HTML(http.StatusOK, "index.html", gin.H{
					"shortened": shortened,
					"host":      Host,
				})
				return
			}

			// Get a new shortend URL
			shortened = newShortenedURL()
			ks.Set(url, shortened)
			ks.Set(shortened, url)
			go jsonstore.Save(ks, "urls.json.gz")
			c.HTML(http.StatusOK, "index.html", gin.H{
				"shortened": shortened,
				"host":      Host,
			})
			return
		} else {
			// Redirect the URL if it is shortened
			var url string
			err := ks.Get(action, &url)
			if err == nil {
				c.Redirect(301, url)
				return
			} else {
				if action == "" {
					c.HTML(http.StatusOK, "index.html", gin.H{"host": Host})
				} else {
					c.HTML(http.StatusOK, "index.html", gin.H{
						"error": "Could not find " + action,
						"host":  Host,
					})
				}
			}
		}
	})
	fmt.Println("Listening on port", Port)
	r.Run(":" + Port) // listen and serve on 0.0.0.0:8080
}

// From http://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
