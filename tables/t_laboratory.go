package tables

const (
	TableLaboratoryFields = "id,name,visitCardId,age,sex,sampleNo,sampleNoTime,diagnosis,projectName,projectResult," +
		"createTime,updateTime"
)

type TLaboratory struct {
	ID            int64  `json:"id" gorm:"column:id"`                       // 自增ID 策略id
	Name          string `json:"name" gorm:"column:name"`                   // 患者姓名
	VisitCardID   string `json:"visitCardId" gorm:"column:visitCardId"`     // 就诊卡ID
	Age           string `json:"age" gorm:"column:age"`                     // 年龄
	Sex           string `json:"sex" gorm:"column:sex"`                     // 性别
	SampleNo      string `json:"sampleNo" gorm:"column:sampleNo"`           // 标本号
	SampleNoTime  int    `json:"sampleNoTime" gorm:"column:sampleNoTime"`   // 时间
	Diagnosis     string `json:"diagnosis" gorm:"column:diagnosis"`         // 诊断
	ProjectName   string `json:"projectName" gorm:"column:projectName"`     // 项目名称
	ProjectResult string `json:"projectResult" gorm:"column:projectResult"` // 项目结果
	CreateTime    string `json:"createTime" gorm:"column:createTime"`       // 记录创建时间
	UpdateTime    string `json:"updateTime" gorm:"column:updateTime"`       // 记录最后更新时间
}
