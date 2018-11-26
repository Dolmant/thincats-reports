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

	"github.com/gocarina/gocsv"
	"github.com/robfig/cron"
	sendgrid "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"

	"net/http/cookiejar"
)

// sub part of https://borrower.thincats.com.au/api/loans/2/exchange/balance-transactions
type BalanceTransaction struct {
	AccruedInterestAmount     float64 `json:"accruedInterestAmount"`
	CapitalizedInterestAmount float64 `json:"capitalizedInterestAmount"`
	FeeAmount                 float64 `json:"feeAmount"`
	PrincipleAmount           float64 `json:"principleAmount"`
	RepaymentDueAmount        float64 `json:"repaymentDueAmount"`
	Total                     float64 `json:"total"`
	Date                      string  `json:"date"`
}

// https://borrower.thincats.com.au/api/loans/2/exchange/balance-transactions
type BalanceTransactions struct {
	AccruedInterestAmount     float64              `json:"accruedInterestAmount"`
	CapitalizedInterestAmount float64              `json:"capitalizedInterestAmount"`
	FeeAmount                 float64              `json:"feeAmount"`
	PrincipleAmount           float64              `json:"principleAmount"` //the principal remaining
	RepaymentDueAmount        float64              `json:"repaymentDueAmount"`
	TotalAmount               float64              `json:"totalAmount"`
	Transactions              []BalanceTransaction `json:"transactions"`
}

// https://borrower.thincats.com.au/api/loans/2/exchange/dues
type Repayment struct {
	BorrowerInvestorIndicator string  `json:"borrowerInvestorIndicator"`
	DemandDueStatus           string  `json:"demandDueStatus"`
	DueAmt                    float64 `json:"dueAmt"`
	DueType                   string  `json:"dueType"`
	DueValueDate              string  `json:"dueValueDate"`
	OrgDueAmt                 float64 `json:"orgDueAmt"`
	FeePerRepayAmt            float64 `json:"feePerRepayAmt"`
	IntCapitalizedFlag        string  `json:"intCapitalizedFlag"`
}

// https://borrower.thincats.com.au/api/loans/2/exchange
// https://borrower.thincats.com.au/api/loans/2/exchange/current-status
type Loan struct {
	Amount                               float64 `json:"amount"` //Original amount
	Created                              string  `json:"created"`
	BorrowerRate                         float64 `json:"borrowerRate"`
	EndDate                              string  `json:"endDate"`
	LoanTerm                             string  `json:"loanTerm"`
	LoanTermMonth                        float64 `json:"loanTermMonth"`
	LoanTermYear                         float64 `json:"loanTermYear"`
	FirstName                            string  `json:"firstName"`
	LastName                             string  `json:"lastName"`
	BorrowerFeePerRepay                  float64 `json:"borrowerFeePerRepay"`
	BorrowerName                         string  `json:"borrowerName"`
	LoanAccountID                        string  `json:"loanAccountId"`
	LoanAcctDueDefaultStatus             string  `json:"loanAcctDueDefaultStatus"`
	LoanAcctIntCapitalizedFlag           string  `json:"loanAcctIntCapitalizedFlag"`
	LoanAcctIntCapitalizedFlagInvestor   string  `json:"loanAcctIntCapitalizedFlagInvestor"`
	LoanAcctStatus                       string  `json:"loanAcctStatus"`
	LoanAmount                           float64 `json:"loanAmount"`
	LoanDescription                      string  `json:"loanDescription"`
	LoanDurationInMonths                 int64   `json:"loanDurationInMonths"`
	LoanGrade                            string  `json:"loanGrade"`
	LoanInterestApplicationDay           int64   `json:"loanInterestApplicationDay"`
	LoanInterestApplicationFrequency     int64   `json:"loanInterestApplicationFrequency"`
	LoanInterestApplicationFrequencyType string  `json:"loanInterestApplicationFrequencyType"`
	LoanInvestorRateOfInterest           float64 `json:"loanInvestorRateOfInterest"`
	LoanLvr                              float64 `json:"loanLvr"`
	LoanPenalRateOfInterest              float64 `json:"loanPenalRateOfInterest"`
	LoanProdCode                         string  `json:"loanProdCode"`
	LoanPurpose                          string  `json:"loanPurpose"`
	LoanRateOfInterest                   float64 `json:"loanRateOfInterest"`
	LoanRepaymentMethod                  string  `json:"loanRepaymentMethod"`
	LoanStartDate                        string  `json:"loanStartDate"`
	NextIntCalcStartDate                 string  `json:"nextIntCalcStartDate"`
	SecurityDetails                      string  `json:"securityDetails"`
	SpdsDocLink                          string  `json:"spdsDocLink"`
	Version                              int64   `json:"version"`
	Email                                string  `json:"email"`
	StartDate                            string  `json:"startDate"`
	StatusName                           string  `json:"statusName"`
	Name                                 string  `json:"name"`
	Id                                   int64   `json:"id"`
	ExchangeLoanId                       string  `json:"exchangeLoanId"`
	BalanceTransactions                  BalanceTransactions
	Repayments                           []Repayment
}

// https://investorapi.thincats.com.au//loans/loan/blk/invportfolio/10240/1/20
// https://investorapi.thincats.com.au//loans/loan/exchange/investor/interest/10240/1/20
// https://investorapi.thincats.com.au//loans/loan/exchange/investor/loans/10240/1/20
type InvestorLoan struct {
	LoanAmount                 float64 `json:"loanAmount"`
	OutstandingPrinciple       float64 `json:"outstandingPrinciple"`
	RepaymentPaid              string  `json:"repaymentPaid"`
	RepaymentsToBePaid         string  `json:"repaymentsToBePaid"`
	NextRepaymentDate          string  `json:"nextRepaymentDate"`
	BorrowerName               string  `json:"borrowerName"`
	LoanStartdate              string  `json:"loanStartdate"`
	LoanEnddate                string  `json:"loanEnddate"`
	NumofTokensHeld            float64 `json:"numofTokensHeld"`
	CurrentTokenValue          float64 `json:"currentTokenValue"`
	OrgTokenValue              float64 `json:"orgTokenValue"`
	TokenAllocationDate        string  `json:"tokenAllocationDate"`
	InterestEarned             float64 `json:"interestEarned"`
	LoanInvestorRateOfInterest float64 `json:"loanInvestorRateOfInterest"`
	LoanAcctId                 string  `json:"loanAcctId"`
	InvestorLoan               *Loan
}

// https://investorapi.thincats.com.au//loans/loan/exchange/10240/txn/1/20
type Transaction struct {
	TxnId                     string  `json:"txnId"`
	TxnValueDate              string  `json:"txnValueDate"`
	TxnAmt                    float64 `json:"txnAmt"`
	Txnind                    string  `json:"txnind"`
	TxnType                   string  `json:"txnType"`
	TxnDesc                   string  `json:"txnDesc"`
	AcctId                    string  `json:"acctId"`
	PartitionKey              string  `json:"partitionKey"`
	BorrowerInvestorIndicator string  `json:"borrowerInvestorIndicator"`
}

// investors
// https://investorapi.thincats.com.au/loans/loan/exchange/investor/balance/10240
type Investor struct {
	AccountName      string  `json:"accountName"`
	Email            string  `json:"email"`
	GivenName        string  `json:"givenName"`
	Surname          string  `json:"surname"`
	Type             string  `json:"type"`
	AccountBalance   float64 `json:"accountBalance"`
	BalanceInHold    float64 `json:"balanceInHold"`
	EffectiveBalance float64 `json:"effectiveBalance"`
	InterestEarned   float64 `json:"interestEarned"`
	// This is the ID effectively
	InvestorAcctName           string         `json:"investorAcctName"`
	NumOfLoansInvested         float64        `json:"numOfLoansInvested"`
	NumOfLoansInvestedInClosed float64        `json:"numOfLoansInvestedInClosed"`
	PrincipalInLiveLoanAccts   float64        `json:"principalInLiveLoanAccts"`
	PrincipleToRecover         float64        `json:"principleToRecover"`
	InvestorLoans              []InvestorLoan `json:"investorPortFolioDtoList"` // todo pagination for these transactions
	Transactions               []Transaction  `json:"transactionLegPageList"`   // todo pagination for these transactions
}

type Config struct {
	Email                string
	Password             string
	Username             string
	Pass                 string
	Basic                string
	SENDGRID_API_KEY     string
	Interval             int64
	SimultaneousRequests int64
	DetailedMatch        bool
}

// https://borrower.thincats.com.au/api/loans/loan-managements
// https://investorapi.thincats.com.au//accounts/accounts
type Data struct {
	Config        Config
	InvestorToken string
	CookieJar     *cookiejar.Jar
	Loans         []Loan
	Investors     []Investor
	Semaphore     chan int
	BMutex        *sync.Mutex
	IMutex        *sync.Mutex
}

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
		type CapitalOutstanding struct {
			LoanId             string `csv:"loan_id"`
			LoanName           string `csv:"loan_name"`
			InvestorName       string `csv:"investor_name"`
			InvestorId         string `csv:"investor_id"`
			CapitalOutstanding string `csv:"capital_outstanding"`
			OriginalCapital    string `csv:"original_capital"`
		}

		CapitalOutstandingTotal := []*CapitalOutstanding{}

		for _, investor := range Data.Investors {
			for _, loan := range investor.InvestorLoans {

				CapitalOutstandingTotal = append(CapitalOutstandingTotal, &CapitalOutstanding{
					InvestorId:         investor.AccountName,
					InvestorName:       investor.GivenName + " " + investor.Surname,
					LoanId:             loan.LoanAcctId,
					LoanName:           loan.BorrowerName,
					CapitalOutstanding: strconv.FormatFloat(float64(loan.CurrentTokenValue*loan.NumofTokensHeld), 'f', 6, 32),
					OriginalCapital:    strconv.FormatFloat(float64(loan.OrgTokenValue*loan.NumofTokensHeld), 'f', 6, 32),
				})
			}
		}

		csvContent, err := gocsv.MarshalString(&CapitalOutstandingTotal)
		if err != nil {
			panic(err)
		}
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=capital-outstanding-report.csv")
		c.Data(http.StatusOK, "text/csv", []byte(csvContent))
	})
	authorized.GET("/report/investor-balance", func(c *gin.Context) {
		type InvestorBalance struct {
			InvestorName             string `csv:"investor_name"`
			InvestorId               string `csv:"investor_id"`
			AccountBalance           string `csv:"accountBalance"`
			BalanceInHold            string `csv:"balanceInHold"`
			EffectiveBalance         string `csv:"effectiveBalance"`
			PrincipalInLiveLoanAccts string `csv:"principalInLiveLoanAccts"`
		}

		InvestorBalanceTotal := []*InvestorBalance{}

		for _, investor := range Data.Investors {
			InvestorBalanceTotal = append(InvestorBalanceTotal, &InvestorBalance{
				InvestorId:               investor.AccountName,
				InvestorName:             investor.GivenName + " " + investor.Surname,
				AccountBalance:           strconv.FormatFloat(float64(investor.AccountBalance), 'f', 6, 32),
				BalanceInHold:            strconv.FormatFloat(float64(investor.BalanceInHold), 'f', 6, 32),
				PrincipalInLiveLoanAccts: strconv.FormatFloat(float64(investor.PrincipalInLiveLoanAccts), 'f', 6, 32),
				EffectiveBalance:         strconv.FormatFloat(float64(investor.EffectiveBalance), 'f', 6, 32),
			})
		}

		csvContent, err := gocsv.MarshalString(&InvestorBalanceTotal)
		if err != nil {
			panic(err)
		}
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=investor-balance-report.csv")
		c.Data(http.StatusOK, "text/csv", []byte(csvContent))
	})
	authorized.GET("/report/investor-transactions", func(c *gin.Context) {
		type InvestorTransactions struct {
			InvestorName              string `csv:"investor_name"`
			InvestorId                string `csv:"investor_id"`
			TxnId                     string `csv:"txnId"`
			TxnValueDate              string `csv:"txnValueDate"`
			TxnAmt                    string `csv:"txnAmt"`
			Txnind                    string `csv:"txnind"`
			TxnType                   string `csv:"txnType"`
			TxnDesc                   string `csv:"txnDesc"`
			AcctId                    string `csv:"acctId"`
			PartitionKey              string `csv:"partitionKey"`
			BorrowerInvestorIndicator string `csv:"borrowerInvestorIndicator"`
		}

		InvestorTransactionsTotal := []*InvestorTransactions{}

		for _, investor := range Data.Investors {
			for _, transaction := range investor.Transactions {
				InvestorTransactionsTotal = append(InvestorTransactionsTotal, &InvestorTransactions{
					InvestorId:                investor.AccountName,
					InvestorName:              investor.GivenName + " " + investor.Surname,
					TxnId:                     transaction.TxnId,
					TxnValueDate:              transaction.TxnValueDate,
					TxnAmt:                    strconv.FormatFloat(float64(transaction.TxnAmt), 'f', 6, 32),
					Txnind:                    transaction.Txnind,
					TxnType:                   transaction.TxnType,
					TxnDesc:                   transaction.TxnDesc,
					AcctId:                    transaction.AcctId,
					PartitionKey:              transaction.PartitionKey,
					BorrowerInvestorIndicator: transaction.BorrowerInvestorIndicator,
				})
			}
		}

		csvContent, err := gocsv.MarshalString(&InvestorTransactionsTotal)
		if err != nil {
			panic(err)
		}
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=investor-transaction-report.csv")
		c.Data(http.StatusOK, "text/csv", []byte(csvContent))
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
