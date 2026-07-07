package dto

type ExpenseRequest struct {
	Category    string `json:"category" binding:"required,oneof=rent utilities supplies salaries other"`
	Description string `json:"description" binding:"required,min=1,max=300"`
	Amount      int64  `json:"amount" binding:"required,min=1"`
	ExpenseDate string `json:"expense_date" binding:"omitempty,datetime=2006-01-02"`
}
