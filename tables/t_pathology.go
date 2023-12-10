package tables

const (
	TablePathologyFields = "id,name,visitCardId,visitTime,pathologyID,admissionNumber,age,sex," +
		"pathologyResult,pathologyTime,createTime,updateTime"
)

type TPathology struct {
	ID              int64  `json:"id" gorm:"column:id"`                           // 自增ID 策略id
	Name            string `json:"name" gorm:"column:name"`                       // 患者姓名
	VisitCardID     string `json:"visitCardId" gorm:"column:visitCardId"`         // 就诊卡ID
	VisitTime       int    `json:"visitTime" gorm:"column:visitTime"`             // 就诊时间
	PathologyID     string `json:"pathologyID" gorm:"column:pathologyID"`         // 病理号
	AdmissionNumber string `json:"admissionNumber" gorm:"column:admissionNumber"` // 住院号
	Age             string `json:"age" gorm:"column:age"`                         // 年龄
	Sex             string `json:"sex" gorm:"column:sex"`                         // 性别
	PathologyResult string `json:"pathologyResult" gorm:"column:pathologyResult"` // 病理结果
	PathologyTime   int    `json:"pathologyTime" gorm:"column:pathologyTime"`     // 肝穿时间 对应主表登机时间
	CreateTime      string `json:"createTime" gorm:"column:createTime"`           // 记录创建时间
	UpdateTime      string `json:"updateTime" gorm:"column:updateTime"`           // 记录最后更新时间
}

func (m *TPathology) TableName() string {
	return "t_pathology"
}
