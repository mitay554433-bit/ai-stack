package adapters

type Adapter struct {
	ID           string
	Capabilities []string
	Enabled      bool
	Authority    string
}

func Catalog(localGemma bool) []Adapter {
	return []Adapter{
		{ID: "LOCAL_GEMMA", Capabilities: []string{"REASON", "ANALYZE", "DRAFT", "SIMULATE"}, Enabled: localGemma, Authority: "CAP_ONLY"},
		{ID: "GITHUB", Capabilities: []string{"PROGRAM", "VERSION", "PATENT_EVIDENCE"}, Authority: "BOUNDED_CAP"},
		{ID: "EMAIL", Capabilities: []string{"READ", "DRAFT", "SEND"}, Authority: "SEND_GATED"},
		{ID: "PAYMENTS", Capabilities: []string{"PRODUCT", "PRICE", "LINK", "RECEIPT", "TRANSFER"}, Authority: "TRANSFER_GATED"},
		{ID: "CRM", Capabilities: []string{"CUSTOMER", "PRODUCT", "LEAD", "SALE", "SUPPORT"}, Authority: "BOUNDED_CAP"},
		{ID: "WEB", Capabilities: []string{"SITE", "STORE", "DEPLOY"}, Authority: "DEPLOY_GATED"},
		{ID: "RESEARCH", Capabilities: []string{"PATENT", "GRANT", "MARKET", "MA"}, Authority: "EVIDENCE_ONLY"},
	}
}
