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
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/gocarina/gocsv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"

	"net/http/cookiejar"
)

// sub part of https://othera-thincats-prod.azurewebsites.net/api/loans/2/exchange/balance-transactions
type BalanceTransaction struct {
	AccruedInterestAmount     float64 `json:"accruedInterestAmount"`
	CapitalizedInterestAmount float64 `json:"capitalizedInterestAmount"`
	FeeAmount                 float64 `json:"feeAmount"`
	PrincipleAmount           float64 `json:"principleAmount"`
	RepaymentDueAmount        float64 `json:"repaymentDueAmount"`
	Total                     float64 `json:"total"`
	Date                      string  `json:"date"`
}

// https://othera-thincats-prod.azurewebsites.net/api/loans/2/exchange/balance-transactions
type BalanceTransactions struct {
	AccruedInterestAmount     float64              `json:"accruedInterestAmount"`
	CapitalizedInterestAmount float64              `json:"capitalizedInterestAmount"`
	FeeAmount                 float64              `json:"feeAmount"`
	PrincipleAmount           float64              `json:"principleAmount"` //the principal remaining
	RepaymentDueAmount        float64              `json:"repaymentDueAmount"`
	TotalAmount               float64              `json:"totalAmount"`
	Transactions              []BalanceTransaction `json:"transactions"`
}

// https://othera-thincats-prod.azurewebsites.net/api/loans/2/exchange/dues
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

// https://othera-thincats-prod.azurewebsites.net/api/loans/2/exchange/current-status
type Loan struct {
	Amount              float64 `json:"amount"` //Original amount
	Created             string  `json:"created"`
	BorrowerRate        float64 `json:"borrowerRate"`
	EndDate             string  `json:"endDate"`
	LoanTerm            string  `json:"loanTerm"`
	LoanTermMonth       float64 `json:"loanTermMonth"`
	LoanTermYear        float64 `json:"loanTermYear"`
	FirstName           string  `json:"firstName"`
	LastName            string  `json:"lastName"`
	Email               string  `json:"email"`
	StartDate           string  `json:"startDate"`
	StatusName          string  `json:"statusName"`
	Name                string  `json:"name"`
	Id                  int64   `json:"id"`
	ExchangeLoanId      string  `json:"exchangeLoanId"`
	BalanceTransactions BalanceTransactions
	Repayments          []Repayment
}

// https://thinapi.blockbond.com.au//loans/loan/blk/invportfolio/10240/1/20
// https://thinapi.blockbond.com.au//loans/loan/exchange/investor/interest/10240/1/20
// https://thinapi.blockbond.com.au//loans/loan/exchange/investor/loans/10240/1/20
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

// https://thinapi.blockbond.com.au//loans/loan/exchange/10240/txn/1/20
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
// https://thinapi.blockbond.com.au/loans/loan/exchange/investor/balance/10240
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
	Interval             int64
	SimultaneousRequests int64
	DetailedMatch        bool
}

// https://othera-thincats-prod.azurewebsites.net/api/loans/loan-managements
// https://thinapi.blockbond.com.au//accounts/accounts
type Data struct {
	Config        Config
	InvestorToken string
	CookieJar     *cookiejar.Jar
	Loans         []Loan
	Investors     []Investor
	Semaphore     chan int
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
	req, _ := http.NewRequest("POST", "https://thinapi.blockbond.com.au/uaa/oauth/token", strings.NewReader(form.Encode()))
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
	req, _ := http.NewRequest("POST", "https://othera-thincats-prod.azurewebsites.net/api/user", b)
	req.Header.Set("Content-Type", "application/json")
	_, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
}

func (data *Data) RefreshInvestors() {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://thinapi.blockbond.com.au//accounts/accounts", nil)
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
	for i, investor := range data.Investors {
		// if i > 2 {
		// 	continue
		// }
		fmt.Printf("%d of %d\n", i, len(data.Investors))
		data.Semaphore <- 1
		go func(accountName string, toUpdate *Investor) {
			client := &http.Client{}
			req, _ := http.NewRequest("GET", "https://thinapi.blockbond.com.au/loans/loan/exchange/investor/balance/"+accountName, nil)
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
			client := &http.Client{}
			req, _ := http.NewRequest("GET", "https://thinapi.blockbond.com.au//loans/loan/exchange/"+accountName+"/txn/1/5000", nil)
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
			client := &http.Client{}
			req, _ := http.NewRequest("GET", "https://thinapi.blockbond.com.au//loans/loan/blk/invportfolio/"+accountName+"/1/5000", nil)
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
	// spew.Dump(data.Investors[2])
}

func (data *Data) RefreshLoanBalances() {
	for i, loan := range data.Loans {
		// if i > 2 {
		// 	continue
		// }
		fmt.Printf("%d of %d\n", i, len(data.Loans))
		id := strconv.FormatInt(int64(loan.Id), 10)
		data.Semaphore <- 1
		go func(accountName string, toUpdate *Loan) {
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://othera-thincats-prod.azurewebsites.net/api/loans/"+accountName+"/exchange/current-status", nil)
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
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://othera-thincats-prod.azurewebsites.net/api/loans/"+accountName+"/exchange/balance-transactions", nil)
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
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://othera-thincats-prod.azurewebsites.net/api/loans/"+accountName+"/exchange/dues", nil)
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
	// spew.Dump(data.Loans[2])
}

func (data *Data) RefreshLoans() {
	client := &http.Client{
		Jar: data.CookieJar,
	}
	req, _ := http.NewRequest("GET", "https://othera-thincats-prod.azurewebsites.net/api/loans/loan-managements", nil)
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

	Data := Data{CookieJar: cookieJar, Config: conf, Semaphore: make(chan int, conf.SimultaneousRequests)}

	go func() {
		go Data.Refresh()
		for _ = range time.Tick(time.Duration(conf.Interval) * time.Minute) {
			go Data.Refresh()
		}
	}()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "success",
		})
	})

	router.GET("/report/capital-outstanding", func(c *gin.Context) {
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
	router.GET("/report/investor-balance", func(c *gin.Context) {
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
	router.GET("/report/investor-transactions", func(c *gin.Context) {
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

	addLog := func(logs *string, line string) {
		log.Println(line)
		newString := *logs + "\n" + line
		logs = &newString
	}
	addErr := func(logs *string, err error) {
		log.Println(err)
		newString := *logs + "\n" + err.Error()
		logs = &newString
	}

	addSpew := func(logs *string, structure interface{}) {
		spew.Dump(structure)
		newString := *logs + "\n" + spew.Sdump(structure)
		logs = &newString
	}

	withinWeek := func(logs *string, time1 string, time2 string) bool {
		layout := "02/01/2006 15:04 AEST"
		time2Parsed, err := time.Parse(layout, time2)
		if err != nil {
			addErr(logs, err)
			return false
		}
		time1Parsed, err := time.Parse(layout, time2)
		if err != nil {
			addErr(logs, err)
			return false
		}
		diff := time1Parsed.Sub(time2Parsed)
		if diff.Hours() < 96 && diff.Hours() > -96 {
			return true
		}
		return false
	}
	within := func(a float64, b float64) bool {
		if (a-b < 1) && (a-b > -1) {
			return true
		}
		return false
	}

	router.GET("/data/loan-balance-conciliation", func(c *gin.Context) {
		// todo if I throw all the data into an id keyed map instead of an array it will be far faster to access
		// match dates
		// split into CSVs

		logs := ""

		loanBalanceRec := 0

		loanBalanceProblemLoans := make(map[string]int64)

		// Loan balance rec
		{
			addLog(&logs, "Loan Balance Rec")
			type CSVLoanBalance struct {
				Id          string  `csv:"Id"`
				Name        string  `csv:"Name"`
				Org         string  `csv:"Org"`
				Int         string  `csv:"Int"`
				Months      string  `csv:"Months"`
				Settled     string  `csv:"Settled"`
				Closed      string  `csv:"Closed"`
				ElapsedMths string  `csv:"Elapsed mths"`
				BalanceMths string  `csv:"Balance mths"`
				Outstanding float64 `csv:"OS"`
			}

			loanBalanceFile, err := os.OpenFile("loan balances 31oct.csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
			if err != nil {
				panic(err)
			}
			defer loanBalanceFile.Close()

			LoanBalance := []*CSVLoanBalance{}

			if err := gocsv.UnmarshalFile(loanBalanceFile, &LoanBalance); err != nil { // Load clients from file
				panic(err)
			}

			for _, loanBalance := range LoanBalance {
				var loan *Loan
				for _, loan2 := range Data.Loans {
					if loanBalance.Id == loan2.ExchangeLoanId {
						loan = &loan2
						break
					}
					if loanBalance.Id == "4956" && loan2.ExchangeLoanId == "L182980532" {
						loan = &loan2
						break
					}
				}
				if loan == nil {
					addLog(&logs, "Could not find a matching loan:")
					addLog(&logs, loanBalance.Id)
					addLog(&logs, loanBalance.Name)
					loanBalanceProblemLoans[loanBalance.Id]++
					loanBalanceRec++
					continue
				}
				if within(loanBalance.Outstanding, loan.BalanceTransactions.PrincipleAmount-loan.BalanceTransactions.RepaymentDueAmount) {
					addLog(&logs, "OK")
					continue
				}
				loanBalanceRec++
				loanBalanceProblemLoans[loanBalance.Id]++
				if conf.DetailedMatch {
					addLog(&logs, "Out By")
					addLog(&logs, strconv.FormatFloat(loanBalance.Outstanding-loan.BalanceTransactions.PrincipleAmount+loan.BalanceTransactions.RepaymentDueAmount, 'f', 6, 64))
					addLog(&logs, "expected: ")
					addSpew(&logs, loanBalance)
					addLog(&logs, "got: ")
					loan.BalanceTransactions.Transactions = []BalanceTransaction{}
					addSpew(&logs, loan.BalanceTransactions)
				} else {
					addLog(&logs, "expected: "+strconv.FormatFloat(loan.BalanceTransactions.PrincipleAmount-loan.BalanceTransactions.RepaymentDueAmount, 'f', 6, 64))
					addLog(&logs, "got: "+strconv.FormatFloat(loanBalance.Outstanding, 'f', 6, 64))
					log.Printf("%d%s%s", loan.Id, loan.Name, strconv.FormatFloat(loan.Amount, 'f', 6, 64))
				}
			}
		}

		addLog(&logs, "loanBalanceRec: "+strconv.FormatInt(int64(loanBalanceRec), 10))

		addLog(&logs, "loanBalanceProblemLoans: ")
		addSpew(&logs, loanBalanceProblemLoans)

		addLog(&logs, "Done")

		path := exPath + string(os.PathSeparator) + "rec.txt"

		ioutil.WriteFile(path, []byte(logs), 0666)

		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=31-Oct-rec-report.txt")
		c.Data(http.StatusOK, "text/csv", []byte(logs))
	})
	router.GET("/reportd/lender-holdings-reconciliation", func(c *gin.Context) {
		// todo if I throw all the data into an id keyed map instead of an array it will be far faster to access
		// match dates
		// split into CSVs

		logs := ""

		lenderHoldingRec := 0
		lenderHoldingRecNotFound := 0

		lenderHoldingProblemLoans := make(map[string]int64)

		// lender holding rec
		{
			addLog(&logs, "Investor Holdings Rec")
			type CSVInvestorHoldings struct {
				UserId             string  `csv:"User ID"`
				BusinessName       string  `csv:"Business Name"`
				LoanId             string  `csv:"App ID"`
				Rate               float64 `csv:"Rate"`
				Amount             float64 `csv:"Amount"`
				CapitalOutstanding float64 `csv:"Capital Outstanding"`
			}

			InvestorHoldingsFile, err := os.OpenFile("lender_holding-31oct18.csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
			if err != nil {
				panic(err)
			}
			defer InvestorHoldingsFile.Close()

			InvestorHoldings := []*CSVInvestorHoldings{}

			if err := gocsv.UnmarshalFile(InvestorHoldingsFile, &InvestorHoldings); err != nil { // Load clients from file
				panic(err)
			}

			for _, investorHolding := range InvestorHoldings {
				var loan *InvestorLoan
				for _, investor := range Data.Investors {
					for _, loan2 := range investor.InvestorLoans {
						if investorHolding.UserId == investor.AccountName && investorHolding.LoanId == loan2.LoanAcctId {
							loan = &loan2
							break
						}
						if investorHolding.LoanId == "4956" && loan2.LoanAcctId == "L182980532" && investorHolding.UserId == investor.AccountName {
							loan = &loan2
							break
						}
					}
					if loan != nil {
						break
					}
				}
				if loan == nil {
					addLog(&logs, "Could not find a match, there are no holdings for this user:")
					addLog(&logs, investorHolding.UserId)
					addLog(&logs, investorHolding.BusinessName)
					addLog(&logs, investorHolding.LoanId)
					lenderHoldingRecNotFound++
					lenderHoldingProblemLoans[investorHolding.LoanId]++
					continue
				}
				if within(investorHolding.CapitalOutstanding, (loan.CurrentTokenValue*loan.NumofTokensHeld)) && within(investorHolding.Amount, (loan.OrgTokenValue*loan.NumofTokensHeld)) {
					addLog(&logs, "OK")
					continue
				}
				lenderHoldingRec++
				lenderHoldingProblemLoans[investorHolding.LoanId]++
				if conf.DetailedMatch {
					addLog(&logs, "Out By")
					addLog(&logs, strconv.FormatFloat(investorHolding.CapitalOutstanding-(loan.CurrentTokenValue*loan.NumofTokensHeld), 'f', 6, 64))
					addLog(&logs, strconv.FormatFloat(investorHolding.Amount-(loan.OrgTokenValue*loan.NumofTokensHeld), 'f', 6, 64))
					addLog(&logs, "expected: ")
					addSpew(&logs, investorHolding)
					addLog(&logs, "got: ")
					addSpew(&logs, loan)
				} else {
					addLog(&logs, "expected (outstanding, org): "+strconv.FormatFloat(investorHolding.CapitalOutstanding, 'f', 6, 64)+" and "+strconv.FormatFloat(investorHolding.Amount, 'f', 6, 64))
					addLog(&logs, "got (outstanding, org): "+strconv.FormatFloat((loan.CurrentTokenValue*loan.NumofTokensHeld), 'f', 6, 64)+" and "+strconv.FormatFloat((loan.OrgTokenValue*loan.NumofTokensHeld), 'f', 6, 64))
				}
			}

		}

		addLog(&logs, "lenderHoldingRec: "+strconv.FormatInt(int64(lenderHoldingRec), 10))
		addLog(&logs, "lenderHoldingRecNotFound: "+strconv.FormatInt(int64(lenderHoldingRecNotFound), 10))

		addLog(&logs, "lenderHoldingProblemLoans: ")
		addSpew(&logs, lenderHoldingProblemLoans)

		addLog(&logs, "Done")

		path := exPath + string(os.PathSeparator) + "rec.txt"

		ioutil.WriteFile(path, []byte(logs), 0666)

		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=31-Oct-rec-report.txt")
		c.Data(http.StatusOK, "text/csv", []byte(logs))
	})
	router.GET("/reports/transactions-reconciliations", func(c *gin.Context) {
		// todo if I throw all the data into an id keyed map instead of an array it will be far faster to access
		// match dates
		// split into CSVs

		type CSVTransaction struct {
			UserId          string  `csv:"User ID"`
			LoanId          string  `csv:"Loan ID"`
			User            string  `csv:"User"`
			Date            string  `csv:"Date"`
			TransactionType string  `csv:"Transaction Type"`
			Dr              float64 `csv:"Dr"`
			Cr              float64 `csv:"Cr"`
			RunningBalance  float64 `csv:"Running Balance"`
		}

		logs := ""

		transactionRec := 0

		problemTxns := make(map[string][]CSVTransaction)

		// transaction rec
		{
			addLog(&logs, "Transaction Rec")

			transactionFile, err := os.OpenFile("loan balances 31oct.csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
			if err != nil {
				panic(err)
			}
			defer transactionFile.Close()

			Transactions := []*CSVTransaction{}

			if err := gocsv.UnmarshalFile(transactionFile, &Transactions); err != nil { // Load clients from file
				panic(err)
			}

			doneTxns := make(map[string]bool)

			for _, transaction := range Transactions {
				for _, investor := range Data.Investors {
					if investor.AccountName != transaction.UserId {
						continue
					}
					for index, investorTransaction := range investor.Transactions {
						if transaction.LoanId == investorTransaction.PartitionKey && !doneTxns[investorTransaction.TxnId] {
							//todo match formula?

							amountMatch := within(investorTransaction.TxnAmt, -transaction.Dr) || within(investorTransaction.TxnAmt, transaction.Cr)
							dateMatch := withinWeek(&logs, investorTransaction.TxnValueDate, transaction.Date)

							match := amountMatch && dateMatch
							if match {
								doneTxns[investorTransaction.TxnId] = true
								if conf.DetailedMatch {
									addLog(&logs, "expected: ")
									addSpew(&logs, transaction)
									addLog(&logs, "got: ")
									addSpew(&logs, investorTransaction)
								} else {
									addLog(&logs, "expected: "+strconv.FormatFloat(float64(transaction.Cr), 'f', 6, 64)+" or "+strconv.FormatFloat(float64(transaction.Dr), 'f', 6, 64))
									addLog(&logs, "got: "+strconv.FormatFloat(investorTransaction.TxnAmt, 'f', 6, 64))
									log.Printf("Loan: %s, User: %s", transaction.LoanId, transaction.UserId)
								}
								break
							}
						}
						if index == (len(investor.Transactions) - 1) {
							transactionRec++
							addLog(&logs, "Couldnt find:")
							addSpew(&logs, transaction)
							problemTxns[transaction.UserId] = append(problemTxns[transaction.UserId], *transaction)
						}
					}
				}
			}
		}

		addLog(&logs, "transactionRec: "+strconv.FormatInt(int64(transactionRec), 10))

		addLog(&logs, "problem Txns by user:")
		addSpew(&logs, problemTxns)

		addLog(&logs, "Done")

		path := exPath + string(os.PathSeparator) + "rec.txt"

		ioutil.WriteFile(path, []byte(logs), 0666)

		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=31-Oct-rec-report.txt")
		c.Data(http.StatusOK, "text/csv", []byte(logs))
	})
	router.GET("/report/investor-balance-reconciliation", func(c *gin.Context) {
		// todo if I throw all the data into an id keyed map instead of an array it will be far faster to access
		// match dates
		// split into CSVs

		logs := ""

		investorBalanceRec := 0

		// Investor balance rec
		{

			addLog(&logs, "Investor Balance Rec")
			type CSVInvestorBalance struct {
				Id              string  `csv:"ID"`
				User            string  `csv:"User"`
				Email           string  `csv:"Email"`
				Balance         float64 `csv:"Balance"`
				Funds           float64 `csv:"Funds (Committed)"`
				InvestmentsLive float64 `csv:"Investments (live)"`
				InvestmentsAllT float64 `csv:"Investments (all time)"`
			}

			investorBalanceFile, err := os.OpenFile("user bals cob 31oct18.csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
			if err != nil {
				panic(err)
			}
			defer investorBalanceFile.Close()

			investorBalance := []*CSVInvestorBalance{}

			if err := gocsv.UnmarshalFile(investorBalanceFile, &investorBalance); err != nil { // Load clients from file
				panic(err)
			}

			for _, investorbalance := range investorBalance {
				var investor *Investor
				for _, investor2 := range Data.Investors {
					if investorbalance.Id == investor2.AccountName {
						investor = &investor2
						break
					}
				}
				if investor == nil {
					investorBalanceRec++
					addLog(&logs, "No match found for: "+investorbalance.Id)
					continue
				}
				if within(investorbalance.Balance+investorbalance.Funds, investor.AccountBalance) {
					addLog(&logs, "OK")
					continue
				}
				investorBalanceRec++
				if conf.DetailedMatch {
					addLog(&logs, "Out By")
					addLog(&logs, strconv.FormatFloat(investorbalance.Balance+investorbalance.Funds-investor.AccountBalance, 'f', 6, 64))
					addLog(&logs, "expected: ")
					addSpew(&logs, investorbalance)
					addLog(&logs, "got: ")
					investor.Transactions = []Transaction{}
					investor.InvestorLoans = []InvestorLoan{}
					addSpew(&logs, investor)
				} else {
					addLog(&logs, "expected: "+strconv.FormatFloat(investor.AccountBalance, 'f', 6, 64))
					addLog(&logs, "got: "+strconv.FormatFloat(investorbalance.Balance, 'f', 6, 64))
					log.Printf("%s%s%s%s%s%s", investor.AccountName, investor.GivenName+" "+investor.Surname, strconv.FormatFloat(investor.AccountBalance, 'f', 6, 64), strconv.FormatFloat(float64(investor.BalanceInHold), 'f', 6, 64), strconv.FormatFloat(float64(investor.PrincipalInLiveLoanAccts), 'f', 6, 64), strconv.FormatFloat(float64(investor.EffectiveBalance), 'f', 6, 64))
				}
			}
		}

		addLog(&logs, "investorBalanceRec: "+strconv.FormatInt(int64(investorBalanceRec), 10))

		addLog(&logs, "Done")

		path := exPath + string(os.PathSeparator) + "rec.txt"

		ioutil.WriteFile(path, []byte(logs), 0666)

		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=31-Oct-investor-balance-report.txt")
		c.Data(http.StatusOK, "text/csv", []byte(logs))
	})

	router.GET("/refresh", func(c *gin.Context) {
		go Data.Refresh()
		c.JSON(200, gin.H{
			"message": "success",
		})
	})

	router.GET("/metrics/totals", func(c *gin.Context) {
		type Totals struct {
			investorCount    int
			loanCount        int
			loanOrgBalance   int
			loanCurrBalance  int
			investorInterest int
			investorBalance  int
		}
		c.JSON(200, gin.H{
			"message": "success",
		})
	})

	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error": "No path found",
		})
	})

	router.Run("0.0.0.0:8079")
}
