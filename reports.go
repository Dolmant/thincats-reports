package main

import (
	"strconv"

	"github.com/gocarina/gocsv"
)

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
