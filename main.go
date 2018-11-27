package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"

	"net/http/cookiejar"
)

func (data *Data) LoginInvestor() {
	type Login struct {
		AccessToken string `json:"access_token"`
	}
	client := &http.Client{}
	form := url.Values{}
	form.Add("grant_type", "password")
	form.Add("username", data.Config.Username)
	form.Add("password", data.Config.Pass)
	req, _ := http.NewRequest("POST", "https://investorapi.thincats.com.au/uaa/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Authorization", data.Config.Basic)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	var result Login

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Fatalln(err)
	}
	data.InvestorToken = result.AccessToken
}

func (data *Data) LoginLender() {
	type Login struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	u := Login{Email: data.Config.Email, Password: data.Config.Password}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(u)
	client := &http.Client{
		Jar: data.CookieJar,
	}
	req, _ := http.NewRequest("POST", "https://borrower.thincats.com.au/api/user", b)
	req.Header.Set("Content-Type", "application/json")
	_, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
}

func (data *Data) RefreshInvestors() {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://investorapi.thincats.com.au//accounts/accounts", nil)
	req.Header.Set("Authorization", "Bearer "+data.InvestorToken)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.NewDecoder(resp.Body).Decode(&data.Investors)
	if err != nil {
		log.Fatalln(err)
	}
	data.RefreshInvestorBalances()
}

func (data *Data) RefreshInvestorBalances() {
	data.IMutex.Lock()
	defer data.IMutex.Unlock()
	var wg sync.WaitGroup
	for i, investor := range data.Investors {
		// if i > 2 {
		// 	continue
		// }
		fmt.Printf("%d of %d\n", i, len(data.Investors))
		data.Semaphore <- 1
		wg.Add(3)

		go func(accountName string, toUpdate *Investor) {
			defer wg.Done()
			client := &http.Client{}
			req, _ := http.NewRequest("GET", "https://investorapi.thincats.com.au/loans/loan/exchange/investor/balance/"+accountName, nil)
			req.Header.Set("Authorization", "Bearer "+data.InvestorToken)
			resp, err := client.Do(req)
			<-data.Semaphore
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				log.Fatalln(err)
			}
		}(investor.AccountName, &data.Investors[i])
		data.Semaphore <- 1
		go func(accountName string, toUpdate *Investor) {
			defer wg.Done()
			client := &http.Client{}
			req, _ := http.NewRequest("GET", "https://investorapi.thincats.com.au//loans/loan/exchange/"+accountName+"/txn/1/5000", nil)
			req.Header.Set("Authorization", "Bearer "+data.InvestorToken)
			resp, err := client.Do(req)
			<-data.Semaphore
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				log.Fatalln(err)
			}
		}(investor.AccountName, &data.Investors[i])
		data.Semaphore <- 1
		go func(accountName string, toUpdate *Investor) {
			defer wg.Done()
			client := &http.Client{}
			req, _ := http.NewRequest("GET", "https://investorapi.thincats.com.au//loans/loan/blk/invportfolio/"+accountName+"/1/5000", nil)
			req.Header.Set("Authorization", "Bearer "+data.InvestorToken)
			resp, err := client.Do(req)
			<-data.Semaphore
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				log.Fatalln(err)
			}
		}(investor.AccountName, &data.Investors[i])
	}
	wg.Wait()
	// spew.Dump(data.Investors[2])
}

func (data *Data) RefreshLoanBalances() {
	data.BMutex.Lock()
	defer data.BMutex.Unlock()
	var wg sync.WaitGroup
	for i, loan := range data.Loans {
		// if i > 2 {
		// 	continue
		// }
		fmt.Printf("%d of %d\n", i, len(data.Loans))
		id := strconv.FormatInt(int64(loan.Id), 10)
		data.Semaphore <- 1
		wg.Add(4)
		go func(accountName string, toUpdate *Loan) {
			defer wg.Done()
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://borrower.thincats.com.au/api/loans/"+accountName+"/exchange/current-status", nil)
			resp, err := client.Do(req)
			<-data.Semaphore
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)
				newStr := buf.String()
				log.Println(newStr)
				log.Fatalln(err)
			}
		}(id, &data.Loans[i])
		data.Semaphore <- 1
		go func(accountName string, toUpdate *Loan) {
			defer wg.Done()
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://borrower.thincats.com.au/api/loans/"+accountName+"/exchange", nil)
			resp, err := client.Do(req)
			<-data.Semaphore
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)
				newStr := buf.String()
				log.Println(newStr)
				log.Fatalln(err)
			}
		}(id, &data.Loans[i])
		data.Semaphore <- 1
		go func(accountName string, toUpdate *BalanceTransactions) {
			defer wg.Done()
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://borrower.thincats.com.au/api/loans/"+accountName+"/exchange/balance-transactions", nil)
			resp, err := client.Do(req)
			<-data.Semaphore
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				log.Fatalln(err)
			}
		}(id, &data.Loans[i].BalanceTransactions)
		data.Semaphore <- 1
		go func(accountName string, toUpdate *[]Repayment) {
			defer wg.Done()
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://borrower.thincats.com.au/api/loans/"+accountName+"/exchange/dues", nil)
			resp, err := client.Do(req)
			<-data.Semaphore
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				log.Fatalln(err)
			}
		}(id, &data.Loans[i].Repayments)
	}
	wg.Wait()
}

func (data *Data) RefreshLoans() {
	client := &http.Client{
		Jar: data.CookieJar,
	}
	req, _ := http.NewRequest("GET", "https://borrower.thincats.com.au/api/loans/loan-managements", nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.NewDecoder(resp.Body).Decode(&data.Loans)
	data.RefreshLoanBalances()
	if err != nil {
		log.Fatalln(err)
	}
}

func (data *Data) Refresh() {
	data.LoginInvestor()
	data.LoginLender()
	go data.RefreshInvestors()
	go data.RefreshLoans()
}

func main() {
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)

	router := gin.New()
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "HEAD"}
	config.AllowHeaders = []string{"Origin", "Authorization", "Content-Length", "Content-Type"}
	router.Use(cors.New(config))
	cookieJar, _ := cookiejar.New(nil)

	byteValue, _ := ioutil.ReadFile(exPath + string(os.PathSeparator) + "config.json")

	// we initialize our Users array
	var conf Config

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &conf)

	Data := Data{
		CookieJar: cookieJar,
		Config:    conf,
		Semaphore: make(chan int, conf.SimultaneousRequests),
		BMutex:    &sync.Mutex{},
		IMutex:    &sync.Mutex{},
	}

	go Data.Refresh()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "success",
		})
	})

	authorized := router.Group("/", gin.BasicAuth(gin.Accounts{
		"Authorization":      Data.Config.Basic,
		Data.Config.Username: Data.Config.Pass,
		Data.Config.Email:    Data.Config.Password,
	}))

	authorized.GET("/report/capital-outstanding", func(c *gin.Context) {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=capital-outstanding-report.csv")
		c.Data(http.StatusOK, "text/csv", []byte(CapitalOutstanding(Data)))
	})

	authorized.GET("/report/investor-balance", func(c *gin.Context) {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=investor-balance-report.csv")
		c.Data(http.StatusOK, "text/csv", []byte(MembershipList(Data)))
	})

	authorized.GET("/report/investor-transactions", func(c *gin.Context) {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=investor-transaction-report.csv")
		c.Data(http.StatusOK, "text/csv", []byte(UserTransactions(Data)))
	})

	authorized.GET("/report/loan-transactions", func(c *gin.Context) {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=loan-transaction-report.csv")
		c.Data(http.StatusOK, "text/csv", []byte(LoanTransactions(Data)))
	})

	authorized.GET("/report/membership-list", func(c *gin.Context) {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=MembershipList-"+time.Now().Format("2006-01-02")+".csv")
		c.Data(http.StatusOK, "text/csv", []byte(MembershipList(Data)))
	})

	authorized.GET("/report/lender-summary", func(c *gin.Context) {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=Lender-Summary-"+time.Now().Format("2006-01-02")+".csv")
		c.Data(http.StatusOK, "text/csv", []byte(LenderSummary(Data)))
	})

	authorized.GET("/report/most-recent-bid-listing", func(c *gin.Context) {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=MostRecentBidListing-"+time.Now().Format("YYYY-MM-DD")+".csv")
		c.Data(http.StatusOK, "text/csv", []byte(BidListing(Data)))
	})

	authorized.GET("/report/loan-loss", func(c *gin.Context) {
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=LoanLoss-"+time.Now().Format("YYYY-MM-DD")+".csv")
		c.Data(http.StatusOK, "text/csv", []byte(LoanLoss(Data)))
	})

	authorized.GET("/refresh", func(c *gin.Context) {
		go Data.Refresh()
		c.JSON(200, gin.H{
			"message": "success",
		})
	})

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error": "No path found",
		})
	})

	//todo check sendgrid worked

	// Data.BMutex.Lock()
	// Data.IMutex.Lock()
	// from := mail.NewEmail("Dylan Simmer", "dsimmer.js@gmail.com")
	// subject := "Daily Reports"
	// to := mail.NewEmail("ThinCats Management", "dsimmer.js@gmail.com") //todo change to management at thincats
	// plainTextContent := "ThinCats automated daily reports"
	// htmlContent := "<strong>ThinCats automated daily reports</strong>"
	// message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	// lsA := mail.NewAttachment()
	// lsA.SetContent(base64.StdEncoding.EncodeToString([]byte(LenderSummary(Data))))
	// lsA.SetType("text/csv")
	// lsA.SetFilename("Lender Summary " + time.Now().Format("2006-01-02") + ".pdf")
	// lsA.SetDisposition("attachment")
	// lsA.SetContentID("Lener Summary")
	// message.AddAttachment(lsA)
	// add the other attachments here

	// client := sendgrid.NewSendClient(conf.SENDGRID_API_KEY)
	// response, err := client.Send(message)
	// if err != nil {
	// 	log.Println(err)
	// } else {
	// 	fmt.Println(response.StatusCode)
	// 	fmt.Println(response.Body)
	// 	fmt.Println(response.Headers)
	// }
	// Data.BMutex.Unlock()
	// Data.IMutex.Unlock()

	//start cron
	c := cron.New()
	c.AddFunc("@every 1h30m", Data.Refresh)
	c.AddFunc("@midnight", func() {
		// todo send email to management at thincats - via sendgrid?
		Data.BMutex.Lock()
		defer Data.BMutex.Unlock()
		Data.IMutex.Lock()
		defer Data.IMutex.Unlock()
		from := mail.NewEmail("Dylan Simmer", "dsimmer.js@gmail.com")
		subject := "Daily Reports"
		to := mail.NewEmail("ThinCats Management", "dsimmer.js@gmail.com") //todo change to management at thincats
		plainTextContent := "ThinCats automated daily reports"
		htmlContent := "<strong>ThinCats automated daily reports</strong>"
		message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
		lsA := mail.NewAttachment()
		lsA.SetContent(LenderSummary(Data))
		lsA.SetType("text/csv")
		lsA.SetFilename("Lender Summary " + time.Now().Format("2006-01-02") + ".pdf")
		lsA.SetDisposition("attachment")
		lsA.SetContentID("Lender Summary")
		message.AddAttachment(lsA)
		// add the other attachments here

		client := sendgrid.NewSendClient(conf.SENDGRID_API_KEY)
		response, err := client.Send(message)
		if err != nil {
			log.Println(err)
		} else {
			fmt.Println(response.StatusCode)
			fmt.Println(response.Body)
			fmt.Println(response.Headers)
		}
	})
	c.Start()
	router.Run("0.0.0.0:8079")
}
