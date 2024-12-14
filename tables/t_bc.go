package tables

const (
	TableBcFields = "id,name,visitCardId,age,sex,visitTime,checkResult,admissionNumber,createTime,updateTime"
)

type TBc struct {
	ID              int64  `json:"id" gorm:"column:id"`                           // 自增ID 策略id
	Name            string `json:"name" gorm:"column:name"`                       // 患者姓名
	VisitCardID     string `json:"visitCardId" gorm:"column:visitCardId"`         // 就诊卡ID
	VisitTime       int    `json:"visitTime" gorm:"column:visitTime"`             // 就诊时间
	Age             string `json:"age" gorm:"column:age"`                         // 年龄
	Sex             string `json:"sex" gorm:"column:sex"`                         // 性别
	CheckResult     string `json:"checkResult" gorm:"column:checkResult"`         // 诊断结果
	AdmissionNumber string `json:"admissionNumber" gorm:"column:admissionNumber"` // 住院号 对应主表来源号
	CreateTime      string `json:"createTime" gorm:"column:createTime"`           // 记录创建时间
	UpdateTime      string `json:"updateTime" gorm:"column:updateTime"`           // 记录最后更新时间
}
