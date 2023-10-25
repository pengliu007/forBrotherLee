package tables

const (
	TableLiverStiffnessFields = "id,name,visitCardId,VisitTime,FiberScansTotalNum,FiberScansSucNum,FatAttenuation," +
		"FatAttenuationQuartileDifference,Hardness,HardnessQuartileDifference,Phone,Height,Weight," +
		"DetectionDuration,SucNums,createTime,updateTime "
)

type TLiverStiffness struct {
	ID                               int64  `json:"id" gorm:"column:id"`                                                             // 自增ID 策略id
	Name                             string `json:"name" gorm:"column:name"`                                                         // 患者姓名
	VisitCardID                      string `json:"visitCardId" gorm:"column:visitCardId"`                                           // 就诊卡ID
	VisitTime                        int    `json:"visitTime" gorm:"column:visitTime"`                                               // 就诊时间
	FiberScansTotalNum               string `json:"fiberScansTotalNum" gorm:"column:fiberScansTotalNum"`                             // 纤维扫描总次数
	FiberScansSucNum                 string `json:"fiberScansSucNum" gorm:"column:fiberScansSucNum"`                                 // 纤维扫描成功次数
	FatAttenuation                   string `json:"fatAttenuation" gorm:"column:fatAttenuation"`                                     // 脂肪衰减(dB/m)
	FatAttenuationQuartileDifference string `json:"fatAttenuationQuartileDifference" gorm:"column:fatAttenuationQuartileDifference"` // 脂肪衰减四分位差(db/m)
	Hardness                         string `json:"hardness" gorm:"column:hardness"`                                                 // 硬度(Kpa)
	HardnessQuartileDifference       string `json:"hardnessQuartileDifference" gorm:"column:hardnessQuartileDifference"`             // 硬度四分位差(Kpa)
	Phone                            string `json:"phone" gorm:"column:phone"`                                                       // 电话
	Height                           string `json:"height" gorm:"column:height"`                                                     // 身高cm
	Weight                           string `json:"weight" gorm:"column:weight"`                                                     // 体重kg
	DetectionDuration                string `json:"detectionDuration" gorm:"column:detectionDuration"`                               // 检测时长（s）
	SucNums                          string `json:"sucNums" gorm:"column:sucNums"`                                                   // 成功次数
	CreateTime                       string `json:"createTime" gorm:"column:createTime"`                                             // 记录创建时间
	UpdateTime                       string `json:"updateTime" gorm:"column:updateTime"`                                             // 记录最后更新时间
}
