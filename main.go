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

	"github.com/gocarina/gocsv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/contrib/gzip"
	"github.com/gin-gonic/gin"

	"net/http/cookiejar"
)

// sub part of https://othera-thincats-prod.azurewebsites.net/api/loans/2/exchange/balance-transactions
type BalanceTransaction struct {
	AccruedInterestAmount     float32 `json:"accruedInterestAmount"`
	CapitalizedInterestAmount float32 `json:"capitalizedInterestAmount"`
	FeeAmount                 float32 `json:"feeAmount"`
	PrincipleAmount           float32 `json:"principleAmount"`
	RepaymentDueAmount        float32 `json:"repaymentDueAmount"`
	Total                     float32 `json:"total"`
	Date                      string  `json:"date"`
}

// https://othera-thincats-prod.azurewebsites.net/api/loans/2/exchange/balance-transactions
type BalanceTransactions struct {
	AccruedInterestAmount     float32              `json:"accruedInterestAmount"`
	CapitalizedInterestAmount float32              `json:"capitalizedInterestAmount"`
	FeeAmount                 float32              `json:"feeAmount"`
	PrincipleAmount           float32              `json:"principleAmount"`
	RepaymentDueAmount        float32              `json:"repaymentDueAmount"`
	TotalAmount               float32              `json:"totalAmount"`
	Transactions              []BalanceTransaction `json:"transactions"`
}

// https://othera-thincats-prod.azurewebsites.net/api/loans/2/exchange/dues
type Repayment struct {
	BorrowerInvestorIndicator string  `json:"borrowerInvestorIndicator"`
	DemandDueStatus           string  `json:"demandDueStatus"`
	DueAmt                    float32 `json:"dueAmt"`
	DueType                   string  `json:"dueType"`
	DueValueDate              string  `json:"dueValueDate"`
	OrgDueAmt                 float32 `json:"orgDueAmt"`
	FeePerRepayAmt            float32 `json:"feePerRepayAmt"`
	IntCapitalizedFlag        string  `json:"intCapitalizedFlag"`
}

// https://othera-thincats-prod.azurewebsites.net/api/loans/2/exchange/current-status
type Loan struct {
	Amount              float32 `json:"amount"`
	Created             string  `json:"created"`
	BorrowerRate        float32 `json:"borrowerRate"`
	EndDate             string  `json:"endDate"`
	LoanTerm            string  `json:"loanTerm"`
	LoanTermMonth       float32 `json:"loanTermMonth"`
	LoanTermYear        float32 `json:"loanTermYear"`
	FirstName           string  `json:"firstName"`
	LastName            string  `json:"lastName"`
	Email               string  `json:"email"`
	StartDate           string  `json:"startDate"`
	StatusName          string  `json:"statusName"`
	Name                string  `json:"name"`
	Id                  int     `json:"id"`
	BalanceTransactions BalanceTransactions
	Repayments          []Repayment
}

// https://thinapi.blockbond.com.au//loans/loan/blk/invportfolio/10240/1/20
// https://thinapi.blockbond.com.au//loans/loan/exchange/investor/interest/10240/1/20
// https://thinapi.blockbond.com.au//loans/loan/exchange/investor/loans/10240/1/20
type InvestorLoan struct {
	LoanAmount                 float32 `json:"loanAmount"`
	OutstandingPrinciple       float32 `json:"outstandingPrinciple"`
	RepaymentPaid              string  `json:"repaymentPaid"`
	RepaymentsToBePaid         string  `json:"repaymentsToBePaid"`
	NextRepaymentDate          string  `json:"nextRepaymentDate"`
	BorrowerName               string  `json:"borrowerName"`
	LoanStartdate              string  `json:"loanStartdate"`
	LoanEnddate                string  `json:"loanEnddate"`
	NumofTokensHeld            float32 `json:"numofTokensHeld"`
	CurrentTokenValue          float32 `json:"currentTokenValue"`
	OrgTokenValue              float32 `json:"orgTokenValue"`
	TokenAllocationDate        string  `json:"tokenAllocationDate"`
	InterestEarned             float32 `json:"interestEarned"`
	LoanInvestorRateOfInterest float32 `json:"loanInvestorRateOfInterest"`
	LoanAcctId                 string  `json:"loanAcctId"`
	InvestorLoan               *Loan
}

// https://thinapi.blockbond.com.au//loans/loan/exchange/10240/txn/1/20
type Transaction struct {
	TxnId                     string  `json:"txnId"`
	TxnValueDate              string  `json:"txnValueDate"`
	TxnAmt                    float32 `json:"txnAmt"`
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
	AccountBalance   float32 `json:"accountBalance"`
	BalanceInHold    float32 `json:"balanceInHold"`
	EffectiveBalance float32 `json:"effectiveBalance"`
	InterestEarned   float32 `json:"interestEarned"`
	// This is the ID effectively
	InvestorAcctName           string         `json:"investorAcctName"`
	NumOfLoansInvested         float32        `json:"numOfLoansInvested"`
	NumOfLoansInvestedInClosed float32        `json:"numOfLoansInvestedInClosed"`
	PrincipalInLiveLoanAccts   float32        `json:"principalInLiveLoanAccts"`
	PrincipleToRecover         float32        `json:"principleToRecover"`
	InvestorLoans              []InvestorLoan `json:"investorPortFolioDtoList"` // todo pagination for these transactions
	Transactions               []Transaction  `json:"transactionLegPageList"`   // todo pagination for these transactions
}

type Config struct {
	Email    string
	Password string
	Username string
	Pass     string
	Basic    string
}

// https://othera-thincats-prod.azurewebsites.net/api/loans/loan-managements
// https://thinapi.blockbond.com.au//accounts/accounts
type Data struct {
	Config        Config
	InvestorToken string
	CookieJar     *cookiejar.Jar
	Loans         []Loan
	Investors     []Investor
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
		func(accountName string, toUpdate *Investor) {
			client := &http.Client{}
			req, _ := http.NewRequest("GET", "https://thinapi.blockbond.com.au/loans/loan/exchange/investor/balance/"+accountName, nil)
			req.Header.Set("Authorization", "Bearer "+data.InvestorToken)
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				log.Fatalln(err)
			}
		}(investor.AccountName, &data.Investors[i])
		func(accountName string, toUpdate *Investor) {
			client := &http.Client{}
			req, _ := http.NewRequest("GET", "https://thinapi.blockbond.com.au//loans/loan/exchange/"+accountName+"/txn/1/5000", nil)
			req.Header.Set("Authorization", "Bearer "+data.InvestorToken)
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				log.Fatalln(err)
			}
		}(investor.AccountName, &data.Investors[i])
		func(accountName string, toUpdate *Investor) {
			client := &http.Client{}
			req, _ := http.NewRequest("GET", "https://thinapi.blockbond.com.au//loans/loan/blk/invportfolio/"+accountName+"/1/5000", nil)
			req.Header.Set("Authorization", "Bearer "+data.InvestorToken)
			resp, err := client.Do(req)
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
		func(accountName string, toUpdate *Loan) {
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://othera-thincats-prod.azurewebsites.net/api/loans/"+accountName+"/exchange/current-status", nil)
			resp, err := client.Do(req)
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
		func(accountName string, toUpdate *BalanceTransactions) {
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://othera-thincats-prod.azurewebsites.net/api/loans/"+accountName+"/exchange/balance-transactions", nil)
			resp, err := client.Do(req)
			if err != nil {
				log.Fatalln(err)
			}

			err = json.NewDecoder(resp.Body).Decode(toUpdate)
			if err != nil {
				log.Fatalln(err)
			}
		}(id, &data.Loans[i].BalanceTransactions)
		func(accountName string, toUpdate *[]Repayment) {
			client := &http.Client{
				Jar: data.CookieJar,
			}
			req, _ := http.NewRequest("GET", "https://othera-thincats-prod.azurewebsites.net/api/loans/"+accountName+"/exchange/dues", nil)
			resp, err := client.Do(req)
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

	Data := Data{CookieJar: cookieJar, Config: conf}

	go func() {
		go Data.Refresh()
		for _ = range time.Tick(15 * time.Minute) {
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
