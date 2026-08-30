package core

import (
	"time"

	"github.com/topbase/topbase/internal/core/queryir"
)

type SemanticTypeOption struct {
	Value     string `json:"value"`
	Label     string `json:"label"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

var SemanticTypeOptions = []SemanticTypeOption{
	{Value: "EntityKey", Label: "主键（实体标识）"},
	{Value: "ForeignKey", Label: "外键"},
	{Value: "Quantity", Label: "数量"},
	{Value: "Score", Label: "评分"},
	{Value: "Percentage", Label: "百分比"},
	{Value: "Currency", Label: "金额（货币）"},
	{Value: "Discount", Label: "折扣"},
	{Value: "Income", Label: "收入"},
	{Value: "Latitude", Label: "纬度"},
	{Value: "Longitude", Label: "经度"},
	{Value: "CreationDate", Label: "创建日期"},
	{Value: "CreationTime", Label: "创建时间"},
	{Value: "CreationTimestamp", Label: "创建日期时间"},
	{Value: "JoinedDate", Label: "加入日期"},
	{Value: "JoinedTime", Label: "加入时间"},
	{Value: "JoinedTimestamp", Label: "加入日期时间"},
	{Value: "Birthday", Label: "生日", Sensitive: true},
	{Value: "EntityName", Label: "实体名称"},
	{Value: "PersonName", Label: "姓名", Sensitive: true},
	{Value: "MobilePhone", Label: "手机号", Sensitive: true},
	{Value: "NationalID", Label: "身份证号", Sensitive: true},
	{Value: "BankCard", Label: "银行卡号", Sensitive: true},
	{Value: "Email", Label: "电子邮箱", Sensitive: true},
	{Value: "Address", Label: "详细地址", Sensitive: true},
	{Value: "URL", Label: "网页地址"},
	{Value: "ImageURL", Label: "图片地址"},
	{Value: "AvatarURL", Label: "头像地址"},
	{Value: "Category", Label: "分类"},
	{Value: "Name", Label: "名称"},
	{Value: "Title", Label: "标题"},
	{Value: "Description", Label: "说明"},
	{Value: "Product", Label: "产品"},
	{Value: "Source", Label: "来源"},
	{Value: "City", Label: "城市"},
	{Value: "State", Label: "省／州"},
	{Value: "Country", Label: "国家／地区"},
	{Value: "ZipCode", Label: "邮政编码"},
	{Value: "IPAddress", Label: "IP 地址", Sensitive: true},
	{Value: "FieldContainingJSON", Label: "JSON 数据"},
}

var SemanticTypes = []string{
	"EntityKey", "ForeignKey",
	"Quantity", "Score", "Percentage", "Currency", "Discount", "Income", "Latitude", "Longitude",
	"CreationDate", "CreationTime", "CreationTimestamp",
	"JoinedDate", "JoinedTime", "JoinedTimestamp", "Birthday",
	"EntityName", "PersonName", "MobilePhone", "NationalID", "BankCard", "Email", "Address", "URL", "ImageURL", "AvatarURL", "Category", "Name", "Title",
	"Description", "Product", "Source", "City", "State", "Country", "ZipCode",
	"IPAddress", "FieldContainingJSON",
}

func ValidSemanticType(value string) bool {
	if value == "" {
		return true
	}
	for _, item := range SemanticTypes {
		if item == value {
			return true
		}
	}
	return false
}

type FieldRef struct {
	Schema string `json:"schema"`
	Table  string `json:"table"`
	Name   string `json:"name"`
}

type FieldMeta struct {
	DatabaseID   string         `json:"database_id"`
	Schema       string         `json:"schema"`
	Table        string         `json:"table"`
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name,omitempty"`
	Description  string         `json:"description,omitempty"`
	SemanticType string         `json:"semantic_type,omitempty"`
	Visibility   string         `json:"visibility,omitempty"`
	Format       map[string]any `json:"format,omitempty"`
	FKTarget     *FieldRef      `json:"fk_target,omitempty"`
}

type ModelColumn struct {
	Name         string `json:"name"`
	DisplayName  string `json:"display_name,omitempty"`
	SemanticType string `json:"semantic_type,omitempty"`
	Description  string `json:"description,omitempty"`
}

type Model struct {
	ID           string         `json:"id"`
	CollectionID string         `json:"collection_id,omitempty"`
	Name         string         `json:"name"`
	DatabaseID   string         `json:"database_id"`
	QueryIR      *queryir.Query `json:"queryir,omitempty"`
	NativeSQL    string         `json:"native_sql,omitempty"`
	Columns      []ModelColumn  `json:"columns,omitempty"`
	CreatedBy    string         `json:"created_by,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

type Metric struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	DatabaseID  string              `json:"database_id"`
	Schema      string              `json:"schema"`
	Table       string              `json:"table"`
	Aggregation queryir.Aggregation `json:"aggregation"`
	Filters     []queryir.Filter    `json:"filters,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
}

type Segment struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	DatabaseID string           `json:"database_id"`
	Schema     string           `json:"schema"`
	Table      string           `json:"table"`
	Filters    []queryir.Filter `json:"filters"`
	CreatedAt  time.Time        `json:"created_at"`
}

type GlossaryTerm struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

type FieldStore interface {
	SaveField(FieldMeta) error
	ListFields(databaseID, schema, table string) ([]FieldMeta, error)
	ListDatabaseFields(databaseID string) ([]FieldMeta, error)
}

type ModelStore interface {
	Create(Model) error
	Update(Model) error
	ByID(id string) (Model, error)
	List() ([]Model, error)
}

type MetricStore interface {
	Create(Metric) error
	ByID(id string) (Metric, error)
	List() ([]Metric, error)
}

type SegmentStore interface {
	Create(Segment) error
	ByID(id string) (Segment, error)
	List() ([]Segment, error)
}

type GlossaryStore interface {
	Create(GlossaryTerm) error
	List() ([]GlossaryTerm, error)
}
