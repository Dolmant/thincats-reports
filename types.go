package main

import (
	"net/http/cookiejar"
	"sync"
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
	MailGunAPIKey        string
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
