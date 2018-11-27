package main

import (
	"strconv"
	"time"

	"github.com/gocarina/gocsv"
)

// Bid Listing gets the morst recent bid listing for each loan
func BidListing(data Data) string {
	csvTitles := "ID,Name,"
	csvBody := ""

	mostRecentLoan := Loan{}
	for _, loan := range data.Loans {
		if loan.StartDate > mostRecentLoan.StartDate {
			mostRecentLoan = loan
		}
	}
	csvTitles += mostRecentLoan.Name + "\n"

	total := float64(0)
	for _, investor := range data.Investors {
		for _, loan := range investor.InvestorLoans {
			if mostRecentLoan.LoanAccountID == loan.LoanAcctId {
				csvBody += investor.AccountName + "," + investor.GivenName + " " + investor.Surname + "," + strconv.FormatFloat(loan.OrgTokenValue*loan.NumofTokensHeld, 'f', 6, 64) + "\n"
				total += loan.OrgTokenValue * loan.NumofTokensHeld
			}
		}
	}
	csvBody += "Total," + strconv.FormatFloat(total, 'f', 6, 64) + "\n"

	return csvTitles + csvBody
}

// LenderSummary gets the lender holdings, the total bids on each loan per investor
func LenderSummary(data Data) string {
	csvTitles := "ID,Name,"
	csvBody := ""
	csvInternal := "INTERNAL,,"
	csvExternal := "EXTERNAL,,"
	csvInternalp := "%,,"
	csvExternalp := "%,,"
	csvTotal := "TOTAL,,"
	csvBidCount := "BIDS,,"
	csvAvgBids := "Avg Bid/Loan,,"

	internalIds := map[string]bool{"10110": true, "10131": true, "10156": true, "10171": true, "10375": true}

	numberInvestments := map[string]float64{}
	internalInvestments := map[string]float64{}
	externalInvestments := map[string]float64{}
	totalInvestments := map[string]float64{}

	var loanIDs []string

	for _, loan := range data.Loans {
		loanIDs = append(loanIDs, loan.LoanAccountID)
		csvTitles += loan.Name + ","
	}
	csvTitles += "Total\n"

	for _, investor := range data.Investors {
		total := float64(0)
		csvBody += investor.AccountName + "," + investor.GivenName + " " + investor.Surname + ","
		for _, loanID := range loanIDs {
			found := false
			for _, loan := range investor.InvestorLoans {
				if loanID == loan.LoanAcctId {
					found = true
					if internalIds[investor.AccountName] {
						internalInvestments[loanID] += loan.OrgTokenValue * loan.NumofTokensHeld
					} else {
						externalInvestments[loanID] += loan.OrgTokenValue * loan.NumofTokensHeld
					}
					numberInvestments[loanID]++
					totalInvestments[loanID] += loan.OrgTokenValue * loan.NumofTokensHeld
					csvBody += strconv.FormatFloat(loan.OrgTokenValue*loan.NumofTokensHeld, 'f', 6, 64) + ","
					total += loan.OrgTokenValue * loan.NumofTokensHeld
				}
			}
			if !found {
				csvBody += ","
			}
		}
		csvBody += strconv.FormatFloat(total, 'f', 6, 64) + "\n"
	}

	totalInt := float64(0)
	totalExt := float64(0)
	total := float64(0)
	totalBids := float64(0)
	for _, loanID := range loanIDs {
		totalInt += internalInvestments[loanID]
		totalExt += externalInvestments[loanID]
		total += totalInvestments[loanID]
		totalBids += numberInvestments[loanID]
		csvInternal += strconv.FormatFloat(internalInvestments[loanID], 'f', 6, 64) + ","
		csvExternal += strconv.FormatFloat(externalInvestments[loanID], 'f', 6, 64) + ","
		csvInternalp += strconv.FormatFloat(internalInvestments[loanID]/totalInvestments[loanID]*100, 'f', 6, 64) + ","
		csvExternalp += strconv.FormatFloat(externalInvestments[loanID]/totalInvestments[loanID]*100, 'f', 6, 64) + ","
		csvTotal += strconv.FormatFloat(totalInvestments[loanID], 'f', 6, 64) + ","
		csvBidCount += strconv.FormatFloat(numberInvestments[loanID], 'f', 6, 64) + ","
		csvAvgBids += strconv.FormatFloat(totalInvestments[loanID]/numberInvestments[loanID], 'f', 6, 64) + ","
	}
	csvInternal += strconv.FormatFloat(totalInt, 'f', 6, 64) + "\n"
	csvExternal += strconv.FormatFloat(totalExt, 'f', 6, 64) + "\n"
	csvInternalp += "," + "\n"
	csvExternalp += "," + "\n"
	csvTotal += strconv.FormatFloat(total, 'f', 6, 64) + "\n"
	csvBidCount += strconv.FormatFloat(totalBids, 'f', 6, 64) + "\n"
	csvAvgBids += "," + "\n"

	return csvTitles + csvInternal + csvInternalp + csvExternal + csvExternalp + csvTotal + csvBidCount + csvAvgBids + csvBody
}

//MembershipList gets the balances for each lender
func MembershipList(data Data) string {
	type InvestorBalance struct {
		InvestorName             string `csv:"Investor Name"`
		InvestorID               string `csv:"Investor ID"`
		AccountBalance           string `csv:"Account Balance"`
		BalanceInHold            string `csv:"Balance Committed"`
		EffectiveBalance         string `csv:"Effective Balance"`
		PrincipalInLiveLoanAccts string `csv:"Principal In Live Loan Accounts"`
	}

	InvestorBalanceTotal := []*InvestorBalance{}

	for _, investor := range data.Investors {
		InvestorBalanceTotal = append(InvestorBalanceTotal, &InvestorBalance{
			InvestorID:               investor.AccountName,
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
	return csvContent
}

// CapitalOutstanding gets said property for each loan
func CapitalOutstanding(data Data) string {
	type CapitalOutstanding struct {
		LoanId             string `csv:"loan_id"`
		LoanName           string `csv:"loan_name"`
		InvestorName       string `csv:"investor_name"`
		InvestorId         string `csv:"investor_id"`
		CapitalOutstanding string `csv:"capital_outstanding"`
		OriginalCapital    string `csv:"original_capital"`
	}

	CapitalOutstandingTotal := []*CapitalOutstanding{}

	for _, investor := range data.Investors {
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
	return csvContent
}

// LoanLoss calculates the loan losses for each loan
// todo add in year on year losses
func LoanLoss(data Data) string {
	// todo add in year by year returns and losses - might require trawling the transactions
	type LoanPaymentSummary struct {
		LoanName           string `csv:"Loan Name"`
		LoanID             string `csv:"Loan ID"`
		DueAmount          string `csv:"Due Amount"`
		OriginalAmount     string `csv:"Original Amount"`
		CapitalOutstanding string `csv:"Principal Outstanding"`
	}

	LoanPaymentSummaryTotal := []*LoanPaymentSummary{}

	for _, loan := range data.Loans {
		dueTotal := float64(0)
		overDue := false
		for _, due := range loan.Repayments {
			dueTotal += due.DueAmt
			if due.DueValueDate < time.Now().Add(-time.Hour*24*90).Format("2006-01-02") {
				overDue = true
			}
		}
		if overDue {
			LoanPaymentSummaryTotal = append(LoanPaymentSummaryTotal, &LoanPaymentSummary{
				LoanName:           loan.Name,
				LoanID:             loan.LoanAccountID,
				DueAmount:          strconv.FormatFloat(dueTotal, 'f', 6, 32),
				OriginalAmount:     strconv.FormatFloat(loan.Amount, 'f', 6, 32),
				CapitalOutstanding: strconv.FormatFloat(loan.BalanceTransactions.PrincipleAmount, 'f', 6, 32),
			})
		}
	}

	csvContent, err := gocsv.MarshalString(&LoanPaymentSummaryTotal)
	if err != nil {
		panic(err)
	}
	return csvContent
}

// LoanTransactions has all loan based transactions
func LoanTransactions(data Data) string {
	type LoanTransactions struct {
		LoanName                  string `csv:"loan_name"`
		LoanID                    string `csv:"loan_id"`
		AccruedInterestAmount     string `csv:"accruedInterestAmount"`
		CapitalizedInterestAmount string `csv:"capitalizedInterestAmount"`
		FeeAmount                 string `csv:"feeAmount"`
		PrincipleAmount           string `csv:"principleAmount"`
		RepaymentDueAmount        string `csv:"repaymentDueAmount"`
		Total                     string `csv:"total"`
		Date                      string `csv:"date"`
	}

	LoanTransactionsAll := []*LoanTransactions{}

	for _, loan := range data.Loans {
		for _, transaction := range loan.BalanceTransactions.Transactions {
			LoanTransactionsAll = append(LoanTransactionsAll, &LoanTransactions{
				LoanName:                  loan.Name,
				LoanID:                    loan.LoanAccountID,
				AccruedInterestAmount:     strconv.FormatFloat(transaction.AccruedInterestAmount, 'f', 6, 32),
				CapitalizedInterestAmount: strconv.FormatFloat(transaction.CapitalizedInterestAmount, 'f', 6, 32),
				FeeAmount:                 strconv.FormatFloat(transaction.FeeAmount, 'f', 6, 32),
				PrincipleAmount:           strconv.FormatFloat(transaction.PrincipleAmount, 'f', 6, 32),
				RepaymentDueAmount:        strconv.FormatFloat(transaction.RepaymentDueAmount, 'f', 6, 32),
				Total:                     strconv.FormatFloat(transaction.Total, 'f', 6, 32),
				Date:                      transaction.Date,
			})
		}
	}

	csvContent, err := gocsv.MarshalString(&LoanTransactionsAll)
	if err != nil {
		panic(err)
	}
	return csvContent
}

// UserTransactions is all user based transactions
func UserTransactions(data Data) string {
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

	for _, investor := range data.Investors {
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
	return csvContent
}
