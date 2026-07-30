package muyu

// ============ Asset Upload ============

type AssetUploadResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
	AssetID        string `json:"assetId,omitempty"`
	Kind           string `json:"kind,omitempty"`
	ExpiresInHours int    `json:"expiresInHours,omitempty"`
}

// ============ Asset Lookup ============

type AssetLookupRequest struct {
	API    string `json:"api"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
}

type AssetLookupResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Found   bool   `json:"found"`
	AssetID string `json:"assetId,omitempty"`
}

// ============ Catalog Query ============

type CatalogResponse struct {
	Success   bool          `json:"success"`
	Message   string        `json:"message,omitempty"`
	UpdatedAt string        `json:"updatedAt"`
	Models    []ModelConfig `json:"models"`
}

type ModelConfig struct {
	Model       string            `json:"model"`
	AssetKinds  []string         `json:"assetKinds"`
	Parameters  []ParameterDef   `json:"parameters"`
	Pricing     []PricingConfig   `json:"pricing"`
}

type ParameterDef struct {
	Key      string   `json:"key"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Default  string   `json:"default,omitempty"`
	Options  []any    `json:"options,omitempty"`
}

type PricingConfig struct {
	Price      string         `json:"price"`
	Parameters []PriceParam   `json:"parameters"`
}

type PriceParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ============ Task Creation ============

type TaskCreateRequest struct {
	API        string                 `json:"api"`
	Model      string                 `json:"model"`
	Prompt     string                 `json:"prompt"`
	AssetIDs   []string               `json:"assetIds,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type TaskCreateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	TaskID  string `json:"taskId,omitempty"`
	Status  string `json:"status,omitempty"`
}

// ============ Task Query ============

type TaskQueryResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	TaskID    string `json:"taskId,omitempty"`
	Status    string `json:"status,omitempty"`
	ResultURL string `json:"resultUrl,omitempty"`
	Progress  string `json:"progress,omitempty"`
}

// ============ Verify Card ============

type VerifyCardRequest struct {
	API string `json:"api"`
}

type VerifyCardResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	API     string `json:"api,omitempty"`
	Points  int    `json:"points,omitempty"`
}
