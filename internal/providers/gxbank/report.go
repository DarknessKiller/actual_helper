package gxbank

type GXReport struct {
	Date        string `csv:"Date"`
	Description string `csv:"Description"`
	Amount      string `csv:"Amount"`
	IsCredit    bool
}
