package models

type ActualBudgetReport struct {
	Account       string
	Date          string
	Payee         string
	Notes         string
	CategoryGroup string
	Category      string
	Amount        string
	SplitAmount   string
	Cleared       string
}
