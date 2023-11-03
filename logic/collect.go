package logic

import (
	"errors"
	"fmt"
	"github.com/pengliu007/forBrotherLee/tables"
	"github.com/tealeg/xlsx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
	"strconv"
	"strings"
	"time"
)

type CollectService struct {
	title []*tables.TCollect
	db    *gorm.DB
}

func NewCollectService() *CollectService {
	return &CollectService{
		title: make([]*tables.TCollect, 0),
	}
}

func (m *CollectService) InitDb() (err error) {

	dsn := "root:root@tcp(127.0.0.1:3306)/inspectionInfo?charset=utf8mb4&parseTime=false&maxAllowedPacket=104857600"
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	//m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("new db failed, 异常 err:%+v", err)
		return err
	}
	// 清空db数据
	//err = m.db.Table(tables.TableCollect).Where("id > ?", "0").Delete(&tables.TCollect{}).Error
	//if err != nil {
	//	fmt.Printf("清空表失败")
	//	return err
	//}
	sqlStr := "truncate table " + tables.TableCollect
	err = m.db.Exec(sqlStr, []interface{}{}...).Error
	if err != nil {
		fmt.Printf("清空总表失败异常")
		return err
	}
	return nil
}

// 读取总表数据
func (m *CollectService) LoadFile(fileName string) (err error) {
	fileHandle, err := xlsx.OpenFile(fileName)
	if err != nil {
		fmt.Printf("汇总表打开失败异常,err:%s", err.Error())
		return err
	}
	fmt.Printf("汇总表打开成功，记录数：%d\n", len(fileHandle.Sheets[0].Rows))
	// 入库
	collectList := make([]*tables.TCollect, 0)
	total := 0
	for i, rowInfo := range fileHandle.Sheets[0].Rows {
		cells := rowInfo.Cells
		collectInfo, err := makeFiledFromCell(cells)
		if err != nil {
			if i > 1 {
				fmt.Printf("makeFiledFromCell [异常]，err：%s\n", err.Error())
			}
			continue
		}
		if i < 2 { //前两行无需入库，存下来输出新表格时用即可
			//if i == 1 { 打印表头 写表字段注释用
			//	for _, cell := range cells {
			//		fmt.Printf("%s\n", cell.Value)
			//	}
			//}
			m.title = append(m.title, collectInfo)
			fmt.Printf("前两行为表头无需入库:%d\n", i)
			continue
		}
		if len(collectInfo.Name) <= 0 && len(collectInfo.VisitCardID) <= 0 {
			fmt.Printf("姓名和就诊卡号均为空，直接过滤[异常]\n")
			continue
		}
		if len(collectInfo.Name) <= 0 || len(collectInfo.VisitCardID) <= 0 {
			fmt.Printf("姓名：%s 或就诊卡号 :%s,为空 异常\n", collectInfo.Name, collectInfo.VisitCardID)
			//continue
		}
		collectList = append(collectList, collectInfo)
		if i > 2 && i%100 == 0 { // 每100条写入一次 并重置切片
			err := m.db.Table(tables.TableCollect).CreateInBatches(collectList, len(collectList)).Error
			if err != nil {
				fmt.Printf("db get 异常 err:%s", err.Error())
				return err
			}
			total += len(collectList)
			//fmt.Printf("入库:%d 条,total:%d\n", len(collectList), total)
			collectList = collectList[0:0]
		}

	}
	if len(collectList) > 0 {
		err := m.db.Table(tables.TableCollect).CreateInBatches(collectList, len(collectList)).Error
		if err != nil {
			fmt.Printf("db get 异常 err:%s", err.Error())
			return err
		}
		total += len(collectList)
		//fmt.Printf("入库:%d 条,total:%d\n", len(collectList), total)
		collectList = collectList[0:0]
	}
	fmt.Printf("汇总表入库成功，记录数：%d\n", total)
	return nil
}

func (m *CollectService) MakeNewCollectExcel(fileNum int) (err error) {
	// 查询满足条件的记录数 并计算每个文件的记录数
	var total int64
	err = m.db.Table(tables.TableCollect).Count(&total).Error
	if err != nil {
		fmt.Printf("查询总表数据库记录数异常 err:%s", err.Error())
		return err
	}
	fmt.Printf("新汇总表记录数：%d，分 %d 个文件[新汇总表_n.xlsx]输出\n", total, fileNum)
	fileHandle := make(map[int]*xlsx.File, 0)
	// 预先打开所有结果文件 并都写入表头和sheet
	for i := 1; i <= fileNum; i++ {
		file := xlsx.NewFile()
		_, err := file.AddSheet("sheet1")
		if err != nil {
			fmt.Printf("AddSheet 异常 err：%s", err.Error())
			return err
		}
		// 写入表头
		if len(m.title) > 0 {
			err = addFileRow(m.title[0], file, " ")
			if err != nil {
				return err
			}
		}
		if len(m.title) > 1 {
			err = addFileRow(m.title[1], file, "就诊时间")
			if err != nil {
				return err
			}
		}
		fileHandle[i] = file
	}
	if len(fileHandle) <= 0 {
		return errors.New("输出结果文件数量非法")
	}
	// 分批取数据 并写入excel，注意切换文件
	numPre := (int(total) / fileNum)
	if int(total)%fileNum != 0 {
		numPre += 1
	}
	fileWritingNum := 1
	alreadyWriteRows := 0
	pageSize := 1000
	pageIndex := 1
	var nowParson string
	nowParsonID := 0
	for {
		dataList := make([]*tables.TCollect, 0)
		// 不要带时间条件，因为之前成功的任务可能是在出库中时发生的，任务结束时间在出库中，时间截断了数据
		err = m.db.Table(tables.TableCollect).Select(tables.TableCollectFields).Order("name asc,visitCardId asc,visitTime asc").
			Limit(pageSize).Offset((pageIndex - 1) * pageSize).Find(&dataList).Error
		if nil != err {
			fmt.Printf("获取数据异常 pageIndex：%d\n", pageIndex)
			return err
		}
		count := len(dataList)
		if count == 0 {
			fmt.Printf("QueryTCollect [%d:%d],count is 0 break", pageIndex, pageSize)
			break
		}
		// 逐条写入对应文件
		for _, dataInfo := range dataList {
			if dataInfo.Name+dataInfo.VisitCardID != nowParson {
				// 相同name和编号的给一个统一编号
				nowParsonID += 1
				nowParson = dataInfo.Name + dataInfo.VisitCardID
				dataInfo.F1 = utils.ToString(nowParsonID)
			} else {
				dataInfo.F1 = ""
			}
			//dataInfo.F1 = utils.ToString(nowParsonID)
			err = addFileRow(dataInfo, fileHandle[fileWritingNum], "")
			if err != nil {
				return err
			}
			alreadyWriteRows++
			if alreadyWriteRows == numPre {
				fileWritingNum++
				alreadyWriteRows = 0
			}
		}
		pageIndex++
		if count < pageSize {
			//fmt.Printf("QueryTCollectbreak!count:%d,pageSize:%d ", count, pageSize)
			break
		}
	}
	// 保存结果文件
	for i, file := range fileHandle {
		fileName := "./新汇总表_" + utils.ToString(i) + ".xlsx"
		err = file.Save(fileName)
		if err != nil {
			fmt.Printf("保存新汇总文件【%s】异常：%s\n", fileName, err.Error())
			return err
		}
	}
	return nil
}

func setCellValue(cell *xlsx.Cell, value string) {
	intValue, err := strconv.Atoi(value)
	if err == nil {
		cell.SetValue(intValue)
		return
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err == nil {
		cell.SetValue(floatValue)
		return
	}
	cell.SetValue(value)
}

// 处理时间字段转义和统一编号问题  稍微做的恶心了点 因为时间想存int，输出时和表头有点点类型冲突，先这么别扭着解决
func addFileRow(dataInfo *tables.TCollect, file *xlsx.File, visitTime string) error {
	if len(file.Sheets) != 1 {
		return errors.New("addFileRow 异常，sheet数量非法")
	}

	sheet := file.Sheets[0]
	row := sheet.AddRow()
	cell := row.AddCell()
	setCellValue(cell, dataInfo.F1)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F2)
	cell = row.AddCell()
	cell.Value = dataInfo.Name
	cell = row.AddCell()
	//if len(dataInfo.VisitTime) > 0 {
	//	VisitTime, err := strconv.Atoi(dataInfo.VisitTime)
	//	if err == nil {
	//		cell.SetDateTimeWithFormat(float64(VisitTime), xlsx.DefaultDateOptions.ExcelTimeFormat)
	//	}
	//}
	if len(visitTime) > 0 {
		setCellValue(cell, visitTime)
	} else {
		cell.SetDateTimeWithFormat(float64(dataInfo.VisitTime), xlsx.DefaultDateOptions.ExcelTimeFormat)
	}

	cell = row.AddCell()
	setCellValue(cell, dataInfo.F5)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F6)
	cell = row.AddCell()
	//setCellValue(cell,dataInfo.F7
	if len(dataInfo.F7) > 0 {
		F7i, err := strconv.Atoi(dataInfo.F7)
		if err == nil {
			cell.SetDateTimeWithFormat(float64(F7i), xlsx.DefaultDateOptions.ExcelTimeFormat)
		}
	}
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F8)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.VisitCardID)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F10)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F11)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F12)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F13)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F14)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F15)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F16)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F17)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F18)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F19)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F20)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F21)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F22)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F23)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F24)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F25)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F26)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F27)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F28)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F29)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F30)
	cell = row.AddCell()
	//setCellValue(cell, dataInfo.F31)
	setCellValue(cell, dataInfo.F31)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F32)
	cell = row.AddCell()
	//setCellValue(cell, dataInfo.F32)
	setCellValue(cell, dataInfo.F33)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F34)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F35)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F36)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F37)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F38)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F39)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F40)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F41)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F42)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F43)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F44)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F45)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F46)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F47)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F48)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F49)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F50)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F51)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F52)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F53)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F54)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F55)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F56)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F57)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F58)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F59)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F60)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F61)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F62)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F63)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F64)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F65)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F66)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F67)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F68)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F69)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F70)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F71)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F72)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F73)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F74)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F75)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F76)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F77)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F78)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F79)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F80)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F81)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F82)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F83)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F84)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F85)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F86)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F87)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F88)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F89)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F90)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F91)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F92)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F93)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F94)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F95)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F96)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F97)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F98)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F99)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F100)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F101)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F102)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F103)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F104)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F105)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F106)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F107)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F108)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F109)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F110)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F111)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F112)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F113)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F114)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F115)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F116)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F117)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F118)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F119)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F120)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F121)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F122)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F123)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F124)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F125)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F126)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F127)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F128)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F129)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F130)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F131)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F132)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F133)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F134)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F135)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F136)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F137)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F138)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F139)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F140)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F141)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F142)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F143)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F144)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F145)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F146)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F147)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F148)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F149)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F150)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F151)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F152)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F153)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F154)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F155)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F156)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F157)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F158)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F159)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F160)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F161)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F162)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F163)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F164)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F165)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F166)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F167)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F168)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F169)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F170)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F171)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F172)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F173)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F174)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F175)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F176)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F177)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F178)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F179)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F180)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F181)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F182)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F183)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F184)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F185)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F186)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F187)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F188)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F189)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F190)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F191)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F192)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F193)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F194)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F195)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F196)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F197)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F198)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F199)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F200)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F201)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F202)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F203)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F204)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F205)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F206)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F207)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F208)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F209)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F210)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F211)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F212)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F213)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F214)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F215)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F216)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F217)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F218)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F219)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F220)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F221)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F222)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F223)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F224)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F225)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F226)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F227)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F228)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F229)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F230)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F231)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F232)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F233)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F234)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F235)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F236)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F237)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F238)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F239)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F240)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F241)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F242)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F243)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F244)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F245)
	cell = row.AddCell()
	setCellValue(cell, dataInfo.F246)
	return nil
}

func makeFiledFromCell(cells []*xlsx.Cell) (*tables.TCollect, error) {
	if len(cells) < 9 {
		fmt.Printf("汇总表记录列数异常：列数%d\n", len(cells))
		return nil, errors.New("列数量异常")
	}

	collectInfo := &tables.TCollect{
		F1:   cells[0].Value,
		F2:   cells[1].Value,
		Name: strings.Trim(cells[2].Value, " "), // 对应f3
		//VisitTime:   visitTime,
		F5:          cells[4].Value,
		F6:          cells[5].Value,
		F7:          cells[6].Value,
		F8:          cells[7].Value,                    // 年龄列数据错误太多
		VisitCardID: strings.Trim(cells[8].Value, " "), // 对应f9

		CreateTime: time.Now().Format("2006-01-02 15:04:05"),
		UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	if len(cells) >= 10 {
		collectInfo.F10 = cells[9].Value
	}
	if len(cells) >= 11 {
		collectInfo.F11 = cells[10].Value
	}
	if len(cells) >= 12 {
		collectInfo.F12 = cells[11].Value
	}
	if len(cells) >= 13 {
		collectInfo.F13 = cells[12].Value
	}
	if len(cells) >= 14 {
		collectInfo.F14 = cells[13].Value
	}
	if len(cells) >= 15 {
		collectInfo.F15 = cells[14].Value
	}
	if len(cells) >= 16 {
		collectInfo.F16 = cells[15].Value
	}
	if len(cells) >= 17 {
		collectInfo.F17 = cells[16].Value
	}
	if len(cells) >= 18 {
		collectInfo.F18 = cells[17].Value
	}
	if len(cells) >= 19 {
		collectInfo.F19 = cells[18].Value
	}
	if len(cells) >= 20 {
		collectInfo.F20 = cells[19].Value
	}
	if len(cells) >= 21 {
		collectInfo.F21 = cells[20].Value
	}
	if len(cells) >= 22 {
		collectInfo.F22 = cells[21].Value
	}
	if len(cells) >= 23 {
		collectInfo.F23 = cells[22].Value
	}
	if len(cells) >= 24 {
		collectInfo.F24 = cells[23].Value
	}
	if len(cells) >= 25 {
		collectInfo.F25 = cells[24].Value
	}
	if len(cells) >= 26 {
		collectInfo.F26 = cells[25].Value
	}
	if len(cells) >= 27 {
		collectInfo.F27 = cells[26].Value
	}
	if len(cells) >= 28 {
		collectInfo.F28 = cells[27].Value
	}
	if len(cells) >= 29 {
		collectInfo.F29 = cells[28].Value
	}
	if len(cells) >= 30 {
		collectInfo.F30 = cells[29].Value
	}
	if len(cells) >= 31 {
		collectInfo.F31 = cells[30].Value
	}
	if len(cells) >= 32 {
		collectInfo.F32 = cells[31].Value
	}
	if len(cells) >= 33 {
		collectInfo.F33 = cells[32].Value
	}
	if len(cells) >= 34 {
		collectInfo.F34 = cells[33].Value
	}
	if len(cells) >= 35 {
		collectInfo.F35 = cells[34].Value
	}
	if len(cells) >= 36 {
		collectInfo.F36 = cells[35].Value
	}
	if len(cells) >= 37 {
		collectInfo.F37 = cells[36].Value
	}
	if len(cells) >= 38 {
		collectInfo.F38 = cells[37].Value
	}
	if len(cells) >= 39 {
		collectInfo.F39 = cells[38].Value
	}
	if len(cells) >= 40 {
		collectInfo.F40 = cells[39].Value
	}
	if len(cells) >= 41 {
		collectInfo.F41 = cells[40].Value
	}
	if len(cells) >= 42 {
		collectInfo.F42 = cells[41].Value
	}
	if len(cells) >= 43 {
		collectInfo.F43 = cells[42].Value
	}
	if len(cells) >= 44 {
		collectInfo.F44 = cells[43].Value
	}
	if len(cells) >= 45 {
		collectInfo.F45 = cells[44].Value
	}
	if len(cells) >= 46 {
		collectInfo.F46 = cells[45].Value
	}
	if len(cells) >= 47 {
		collectInfo.F47 = cells[46].Value
	}
	if len(cells) >= 48 {
		collectInfo.F48 = cells[47].Value
	}
	if len(cells) >= 49 {
		collectInfo.F49 = cells[48].Value
	}
	if len(cells) >= 50 {
		collectInfo.F50 = cells[49].Value
	}
	if len(cells) >= 51 {
		collectInfo.F51 = cells[50].Value
	}
	if len(cells) >= 52 {
		collectInfo.F52 = cells[51].Value
	}
	if len(cells) >= 53 {
		collectInfo.F53 = cells[52].Value
	}
	if len(cells) >= 54 {
		collectInfo.F54 = cells[53].Value
	}
	if len(cells) >= 55 {
		collectInfo.F55 = cells[54].Value
	}
	if len(cells) >= 56 {
		collectInfo.F56 = cells[55].Value
	}
	if len(cells) >= 57 {
		collectInfo.F57 = cells[56].Value
	}
	if len(cells) >= 58 {
		collectInfo.F58 = cells[57].Value
	}
	if len(cells) >= 59 {
		collectInfo.F59 = cells[58].Value
	}
	if len(cells) >= 60 {
		collectInfo.F60 = cells[59].Value
	}
	if len(cells) >= 61 {
		collectInfo.F61 = cells[60].Value
	}
	if len(cells) >= 62 {
		collectInfo.F62 = cells[61].Value
	}
	if len(cells) >= 63 {
		collectInfo.F63 = cells[62].Value
	}
	if len(cells) >= 64 {
		collectInfo.F64 = cells[63].Value
	}
	if len(cells) >= 65 {
		collectInfo.F65 = cells[64].Value
	}
	if len(cells) >= 66 {
		collectInfo.F66 = cells[65].Value
	}
	if len(cells) >= 67 {
		collectInfo.F67 = cells[66].Value
	}
	if len(cells) >= 68 {
		collectInfo.F68 = cells[67].Value
	}
	if len(cells) >= 69 {
		collectInfo.F69 = cells[68].Value
	}
	if len(cells) >= 70 {
		collectInfo.F70 = cells[69].Value
	}
	if len(cells) >= 71 {
		collectInfo.F71 = cells[70].Value
	}
	if len(cells) >= 72 {
		collectInfo.F72 = cells[71].Value
	}
	if len(cells) >= 73 {
		collectInfo.F73 = cells[72].Value
	}
	if len(cells) >= 74 {
		collectInfo.F74 = cells[73].Value
	}
	if len(cells) >= 75 {
		collectInfo.F75 = cells[74].Value
	}
	if len(cells) >= 76 {
		collectInfo.F76 = cells[75].Value
	}
	if len(cells) >= 77 {
		collectInfo.F77 = cells[76].Value
	}
	if len(cells) >= 78 {
		collectInfo.F78 = cells[77].Value
	}
	if len(cells) >= 79 {
		collectInfo.F79 = cells[78].Value
	}
	if len(cells) >= 80 {
		collectInfo.F80 = cells[79].Value
	}
	if len(cells) >= 81 {
		collectInfo.F81 = cells[80].Value
	}
	if len(cells) >= 82 {
		collectInfo.F82 = cells[81].Value
	}
	if len(cells) >= 83 {
		collectInfo.F83 = cells[82].Value
	}
	if len(cells) >= 84 {
		collectInfo.F84 = cells[83].Value
	}
	if len(cells) >= 85 {
		collectInfo.F85 = cells[84].Value
	}
	if len(cells) >= 86 {
		collectInfo.F86 = cells[85].Value
	}
	if len(cells) >= 87 {
		collectInfo.F87 = cells[86].Value
	}
	if len(cells) >= 88 {
		collectInfo.F88 = cells[87].Value
	}
	if len(cells) >= 89 {
		collectInfo.F89 = cells[88].Value
	}
	if len(cells) >= 90 {
		collectInfo.F90 = cells[89].Value
	}
	if len(cells) >= 91 {
		collectInfo.F91 = cells[90].Value
	}
	if len(cells) >= 92 {
		collectInfo.F92 = cells[91].Value
	}
	if len(cells) >= 93 {
		collectInfo.F93 = cells[92].Value
	}
	if len(cells) >= 94 {
		collectInfo.F94 = cells[93].Value
	}
	if len(cells) >= 95 {
		collectInfo.F95 = cells[94].Value
	}
	if len(cells) >= 96 {
		collectInfo.F96 = cells[95].Value
	}
	if len(cells) >= 97 {
		collectInfo.F97 = cells[96].Value
	}
	if len(cells) >= 98 {
		collectInfo.F98 = cells[97].Value
	}
	if len(cells) >= 99 {
		collectInfo.F99 = cells[98].Value
	}
	if len(cells) >= 100 {
		collectInfo.F100 = cells[99].Value
	}
	if len(cells) >= 101 {
		collectInfo.F101 = cells[100].Value
	}
	if len(cells) >= 102 {
		collectInfo.F102 = cells[101].Value
	}
	if len(cells) >= 103 {
		collectInfo.F103 = cells[102].Value
	}
	if len(cells) >= 104 {
		collectInfo.F104 = cells[103].Value
	}
	if len(cells) >= 105 {
		collectInfo.F105 = cells[104].Value
	}
	if len(cells) >= 106 {
		collectInfo.F106 = cells[105].Value
	}
	if len(cells) >= 107 {
		collectInfo.F107 = cells[106].Value
	}
	if len(cells) >= 108 {
		collectInfo.F108 = cells[107].Value
	}
	if len(cells) >= 109 {
		collectInfo.F109 = cells[108].Value
	}
	if len(cells) >= 110 {
		collectInfo.F110 = cells[109].Value
	}
	if len(cells) >= 111 {
		collectInfo.F111 = cells[110].Value
	}
	if len(cells) >= 112 {
		collectInfo.F112 = cells[111].Value
	}
	if len(cells) >= 113 {
		collectInfo.F113 = cells[112].Value
	}
	if len(cells) >= 114 {
		collectInfo.F114 = cells[113].Value
	}
	if len(cells) >= 115 {
		collectInfo.F115 = cells[114].Value
	}
	if len(cells) >= 116 {
		collectInfo.F116 = cells[115].Value
	}
	if len(cells) >= 117 {
		collectInfo.F117 = cells[116].Value
	}
	if len(cells) >= 118 {
		collectInfo.F118 = cells[117].Value
	}
	if len(cells) >= 119 {
		collectInfo.F119 = cells[118].Value
	}
	if len(cells) >= 120 {
		collectInfo.F120 = cells[119].Value
	}
	if len(cells) >= 121 {
		collectInfo.F121 = cells[120].Value
	}
	if len(cells) >= 122 {
		collectInfo.F122 = cells[121].Value
	}
	if len(cells) >= 123 {
		collectInfo.F123 = cells[122].Value
	}
	if len(cells) >= 124 {
		collectInfo.F124 = cells[123].Value
	}
	if len(cells) >= 125 {
		collectInfo.F125 = cells[124].Value
	}
	if len(cells) >= 126 {
		collectInfo.F126 = cells[125].Value
	}
	if len(cells) >= 127 {
		collectInfo.F127 = cells[126].Value
	}
	if len(cells) >= 128 {
		collectInfo.F128 = cells[127].Value
	}
	if len(cells) >= 129 {
		collectInfo.F129 = cells[128].Value
	}
	if len(cells) >= 130 {
		collectInfo.F130 = cells[129].Value
	}
	if len(cells) >= 131 {
		collectInfo.F131 = cells[130].Value
	}
	if len(cells) >= 132 {
		collectInfo.F132 = cells[131].Value
	}
	if len(cells) >= 133 {
		collectInfo.F133 = cells[132].Value
	}
	if len(cells) >= 134 {
		collectInfo.F134 = cells[133].Value
	}
	if len(cells) >= 135 {
		collectInfo.F135 = cells[134].Value
	}
	if len(cells) >= 136 {
		collectInfo.F136 = cells[135].Value
	}
	if len(cells) >= 137 {
		collectInfo.F137 = cells[136].Value
	}
	if len(cells) >= 138 {
		collectInfo.F138 = cells[137].Value
	}
	if len(cells) >= 139 {
		collectInfo.F139 = cells[138].Value
	}
	if len(cells) >= 140 {
		collectInfo.F140 = cells[139].Value
	}
	if len(cells) >= 141 {
		collectInfo.F141 = cells[140].Value
	}
	if len(cells) >= 142 {
		collectInfo.F142 = cells[141].Value
	}
	if len(cells) >= 143 {
		collectInfo.F143 = cells[142].Value
	}
	if len(cells) >= 144 {
		collectInfo.F144 = cells[143].Value
	}
	if len(cells) >= 145 {
		collectInfo.F145 = cells[144].Value
	}
	if len(cells) >= 146 {
		collectInfo.F146 = cells[145].Value
	}
	if len(cells) >= 147 {
		collectInfo.F147 = cells[146].Value
	}
	if len(cells) >= 148 {
		collectInfo.F148 = cells[147].Value
	}
	if len(cells) >= 149 {
		collectInfo.F149 = cells[148].Value
	}
	if len(cells) >= 150 {
		collectInfo.F150 = cells[149].Value
	}
	if len(cells) >= 151 {
		collectInfo.F151 = cells[150].Value
	}
	if len(cells) >= 152 {
		collectInfo.F152 = cells[151].Value
	}
	if len(cells) >= 153 {
		collectInfo.F153 = cells[152].Value
	}
	if len(cells) >= 154 {
		collectInfo.F154 = cells[153].Value
	}
	if len(cells) >= 155 {
		collectInfo.F155 = cells[154].Value
	}
	if len(cells) >= 156 {
		collectInfo.F156 = cells[155].Value
	}
	if len(cells) >= 157 {
		collectInfo.F157 = cells[156].Value
	}
	if len(cells) >= 158 {
		collectInfo.F158 = cells[157].Value
	}
	if len(cells) >= 159 {
		collectInfo.F159 = cells[158].Value
	}
	if len(cells) >= 160 {
		collectInfo.F160 = cells[159].Value
	}
	if len(cells) >= 161 {
		collectInfo.F161 = cells[160].Value
	}
	if len(cells) >= 162 {
		collectInfo.F162 = cells[161].Value
	}
	if len(cells) >= 163 {
		collectInfo.F163 = cells[162].Value
	}
	if len(cells) >= 164 {
		collectInfo.F164 = cells[163].Value
	}
	if len(cells) >= 165 {
		collectInfo.F165 = cells[164].Value
	}
	if len(cells) >= 166 {
		collectInfo.F166 = cells[165].Value
	}
	if len(cells) >= 167 {
		collectInfo.F167 = cells[166].Value
	}
	if len(cells) >= 168 {
		collectInfo.F168 = cells[167].Value
	}
	if len(cells) >= 169 {
		collectInfo.F169 = cells[168].Value
	}
	if len(cells) >= 170 {
		collectInfo.F170 = cells[169].Value
	}
	if len(cells) >= 171 {
		collectInfo.F171 = cells[170].Value
	}
	if len(cells) >= 172 {
		collectInfo.F172 = cells[171].Value
	}
	if len(cells) >= 173 {
		collectInfo.F173 = cells[172].Value
	}
	if len(cells) >= 174 {
		collectInfo.F174 = cells[173].Value
	}
	if len(cells) >= 175 {
		collectInfo.F175 = cells[174].Value
	}
	if len(cells) >= 176 {
		collectInfo.F176 = cells[175].Value
	}
	if len(cells) >= 177 {
		collectInfo.F177 = cells[176].Value
	}
	if len(cells) >= 178 {
		collectInfo.F178 = cells[177].Value
	}
	if len(cells) >= 179 {
		collectInfo.F179 = cells[178].Value
	}
	if len(cells) >= 180 {
		collectInfo.F180 = cells[179].Value
	}
	if len(cells) >= 181 {
		collectInfo.F181 = cells[180].Value
	}
	if len(cells) >= 182 {
		collectInfo.F182 = cells[181].Value
	}
	if len(cells) >= 183 {
		collectInfo.F183 = cells[182].Value
	}
	if len(cells) >= 184 {
		collectInfo.F184 = cells[183].Value
	}
	if len(cells) >= 185 {
		collectInfo.F185 = cells[184].Value
	}
	if len(cells) >= 186 {
		collectInfo.F186 = cells[185].Value
	}
	if len(cells) >= 187 {
		collectInfo.F187 = cells[186].Value
	}
	if len(cells) >= 188 {
		collectInfo.F188 = cells[187].Value
	}
	if len(cells) >= 189 {
		collectInfo.F189 = cells[188].Value
	}
	if len(cells) >= 190 {
		collectInfo.F190 = cells[189].Value
	}
	if len(cells) >= 191 {
		collectInfo.F191 = cells[190].Value
	}
	if len(cells) >= 192 {
		collectInfo.F192 = cells[191].Value
	}
	if len(cells) >= 193 {
		collectInfo.F193 = cells[192].Value
	}
	if len(cells) >= 194 {
		collectInfo.F194 = cells[193].Value
	}
	if len(cells) >= 195 {
		collectInfo.F195 = cells[194].Value
	}
	if len(cells) >= 196 {
		collectInfo.F196 = cells[195].Value
	}
	if len(cells) >= 197 {
		collectInfo.F197 = cells[196].Value
	}
	if len(cells) >= 198 {
		collectInfo.F198 = cells[197].Value
	}
	if len(cells) >= 199 {
		collectInfo.F199 = cells[198].Value
	}
	if len(cells) >= 200 {
		collectInfo.F200 = cells[199].Value
	}
	if len(cells) >= 201 {
		collectInfo.F201 = cells[200].Value
	}
	if len(cells) >= 202 {
		collectInfo.F202 = cells[201].Value
	}
	if len(cells) >= 203 {
		collectInfo.F203 = cells[202].Value
	}
	if len(cells) >= 204 {
		collectInfo.F204 = cells[203].Value
	}
	if len(cells) >= 205 {
		collectInfo.F205 = cells[204].Value
	}
	if len(cells) >= 206 {
		collectInfo.F206 = cells[205].Value
	}
	if len(cells) >= 207 {
		collectInfo.F207 = cells[206].Value
	}
	if len(cells) >= 208 {
		collectInfo.F208 = cells[207].Value
	}
	if len(cells) >= 209 {
		collectInfo.F209 = cells[208].Value
	}
	if len(cells) >= 210 {
		collectInfo.F210 = cells[209].Value
	}
	if len(cells) >= 211 {
		collectInfo.F211 = cells[210].Value
	}
	if len(cells) >= 212 {
		collectInfo.F212 = cells[211].Value
	}
	if len(cells) >= 213 {
		collectInfo.F213 = cells[212].Value
	}
	if len(cells) >= 214 {
		collectInfo.F214 = cells[213].Value
	}
	if len(cells) >= 215 {
		collectInfo.F215 = cells[214].Value
	}
	if len(cells) >= 216 {
		collectInfo.F216 = cells[215].Value
	}
	if len(cells) >= 217 {
		collectInfo.F217 = cells[216].Value
	}
	if len(cells) >= 218 {
		collectInfo.F218 = cells[217].Value
	}
	if len(cells) >= 219 {
		collectInfo.F219 = cells[218].Value
	}
	if len(cells) >= 220 {
		collectInfo.F220 = cells[219].Value
	}
	if len(cells) >= 221 {
		collectInfo.F221 = cells[220].Value
	}
	if len(cells) >= 222 {
		collectInfo.F222 = cells[221].Value
	}
	if len(cells) >= 223 {
		collectInfo.F223 = cells[222].Value
	}
	if len(cells) >= 224 {
		collectInfo.F224 = cells[223].Value
	}
	if len(cells) >= 225 {
		collectInfo.F225 = cells[224].Value
	}
	if len(cells) >= 226 {
		collectInfo.F226 = cells[225].Value
	}
	if len(cells) >= 227 {
		collectInfo.F227 = cells[226].Value
	}
	if len(cells) >= 228 {
		collectInfo.F228 = cells[227].Value
	}
	if len(cells) >= 229 {
		collectInfo.F229 = cells[228].Value
	}
	if len(cells) >= 230 {
		collectInfo.F230 = cells[229].Value
	}
	if len(cells) >= 231 {
		collectInfo.F231 = cells[230].Value
	}
	if len(cells) >= 232 {
		collectInfo.F232 = cells[231].Value
	}
	if len(cells) >= 233 {
		collectInfo.F233 = cells[232].Value
	}
	if len(cells) >= 234 {
		collectInfo.F234 = cells[233].Value
	}
	if len(cells) >= 235 {
		collectInfo.F235 = cells[234].Value
	}
	if len(cells) >= 236 {
		collectInfo.F236 = cells[235].Value
	}
	if len(cells) >= 237 {
		collectInfo.F237 = cells[236].Value
	}
	if len(cells) >= 238 {
		collectInfo.F238 = cells[237].Value
	}
	if len(cells) >= 239 {
		collectInfo.F239 = cells[238].Value
	}
	if len(cells) >= 240 {
		collectInfo.F240 = cells[239].Value
	}
	if len(cells) >= 241 {
		collectInfo.F241 = cells[240].Value
	}
	if len(cells) >= 242 {
		collectInfo.F242 = cells[241].Value
	}
	if len(cells) >= 243 {
		collectInfo.F243 = cells[242].Value
	}
	if len(cells) >= 244 {
		collectInfo.F244 = cells[243].Value
	}
	if len(cells) >= 245 {
		collectInfo.F245 = cells[244].Value
	}
	if len(cells) >= 246 {
		collectInfo.F246 = cells[245].Value
	}
	visitTimeInt, err := strconv.Atoi(strings.Trim(cells[3].Value, " "))
	if err != nil {
		visitTimeFloat, err := strconv.ParseFloat(strings.Trim(cells[3].Value, " "), 64)
		if err != nil {
			fmt.Printf("汇总表记录就诊时间异常：姓名:%s，卡号:%s,时间【%s】\n",
				cells[2].Value, cells[8].Value, cells[3].Value)
		} else {
			collectInfo.VisitTime = int(visitTimeFloat)
		}
	} else {
		collectInfo.VisitTime = visitTimeInt
	}

	//c3, err := cells[3].GetTime(false)
	//if err == nil {
	//	collectInfo.F4 = c3.Format("2006-01-02")
	//}
	//c6, err := cells[6].GetTime(false)
	//if err == nil {
	//	collectInfo.F7 = c6.Format("2006-01-02")
	//}

	return collectInfo, nil
}
