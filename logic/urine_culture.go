package logic

import (
	"fmt"
	"github.com/pengliu007/forBrotherLee/tables"
	"github.com/tealeg/xlsx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strings"
	"time"
)

// 尿培养
type UrineCultureService struct {
	mergeAdd      int
	mergeUpdate   int
	mergeConflict int
	mergeErr      int
	db            *gorm.DB
}

type DataToMerge struct {
	ID          int64
	visitCardID string
	name        string
	age         string
	sex         string
	diagnosis   string
	beginTime   int
	first       string
	second      string
	third       string
}

func NewUrineCultureService() *UrineCultureService {
	return &UrineCultureService{}
}

func (m *UrineCultureService) InitDb() (err error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/inspectionInfo?charset=utf8mb4&parseTime=false&maxAllowedPacket=104857600"
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	//m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("new db 异常 failed, err:%+v", err)
		return err
	}
	sqlStr := "truncate table " + tables.TableUrineCulture
	err = m.db.Exec(sqlStr, []interface{}{}...).Error
	if err != nil {
		fmt.Printf("清空尿培养表失败异常")
		return err
	}
	return nil
}

func (m *UrineCultureService) LoadFile(fileName string) (err error) {
	fileHandle, err := xlsx.OpenFile(fileName)
	if err != nil {
		fmt.Printf("尿培养文件打开失败异常,err:%s", err.Error())
		return err
	}
	fmt.Printf("尿培养文件表打开成功，记录数：%d\n", len(fileHandle.Sheets[0].Rows))

	// 入库
	dataList := make([]*tables.TUrineCulture, 0)
	total := 0
	existSampleNo := make(map[string]bool, 0)
	for i, rowInfo := range fileHandle.Sheets[0].Rows {
		if i < 1 { //前1行无需入库
			fmt.Printf("前1行为表头无需入库:%d\n", i)
			continue
		}
		cells := rowInfo.Cells
		if len(cells) < 14 {
			fmt.Printf("cells [异常] len[%d]，err\n", len(cells))
			continue
		}
		if strings.Trim(cells[10].Value, " ") != "尿液" {
			//fmt.Printf("cells [deal_error] type[%s]，非尿检 过滤\n", cells[10])
			continue
		}
		// 相同标本号只入库一条
		if _, ok := existSampleNo[cells[1].Value]; ok {
			//fmt.Printf("existSampleNo[%s]，重复过滤\n", cells[1])
			continue
		}
		existSampleNo[cells[1].Value] = true
		// 时间转换为excel时间
		var excelTime float64
		excelTimeStr := strings.Trim(cells[2].Value, " ")
		if len(excelTimeStr) == 8 {
			sampleNoTime, err := time.Parse("20060102", excelTimeStr)
			if err != nil {
				fmt.Printf("标本时间=%s[异常],err:%s\n", excelTimeStr, err.Error())
				continue
			} else {
				excelTime = xlsx.TimeToExcelTime(sampleNoTime, false)
			}
		} else {
			fmt.Printf("标本时间长度异常[异常]i=%d,value=%s, len:%d\n", i, excelTimeStr, len(excelTimeStr))
			continue
		}
		dataInfo := &tables.TUrineCulture{
			Name:            strings.Trim(cells[5].Value, " "),
			VisitCardID:     strings.Trim(cells[4].Value, " "),
			Age:             strings.Replace(strings.Trim(cells[7].Value, " "), "岁", "", -1),
			Sex:             strings.Trim(cells[6].Value, " "),
			SampleNo:        cells[1].Value,
			SampleNoTime:    int(excelTime),
			MicroDataIDName: cells[13].Value,
			Diagnosis:       cells[9].Value,
			SampleType:      cells[10].Value,
			CreateTime:      time.Now().Format("2006-01-02 15:04:05"),
			UpdateTime:      time.Now().Format("2006-01-02 15:04:05"),
		}
		if len(dataInfo.Name) <= 0 && len(dataInfo.VisitCardID) <= 0 {
			fmt.Printf("姓名和就诊卡号均为空，直接过滤 [deal_error]\n")
			continue
		}
		if len(dataInfo.Name) <= 0 || len(dataInfo.VisitCardID) <= 0 {
			fmt.Printf("姓名：%s 或就诊卡号 :%s,为空\n", dataInfo.Name, dataInfo.VisitCardID)
			//continue
		}
		dataList = append(dataList, dataInfo)
		if i > 2 && i%100 == 0 { // 每100条写入一次 并重置切片
			err := m.db.Table(tables.TableUrineCulture).CreateInBatches(dataList, len(dataList)).Error
			if err != nil {
				fmt.Printf("db get 异常 err:%s", err.Error())
				return err
			}
			total += len(dataList)
			//fmt.Printf("入库:%d 条,total:%d\n", len(collectList), total)
			dataList = dataList[0:0]
		}

	}
	if len(dataList) > 0 {
		err := m.db.Table(tables.TableUrineCulture).CreateInBatches(dataList, len(dataList)).Error
		if err != nil {
			fmt.Printf("db get异常 err:%s", err.Error())
			return err
		}
		total += len(dataList)
		//fmt.Printf("入库:%d 条,total:%d\n", len(collectList), total)
		dataList = dataList[0:0]
	}
	fmt.Printf("尿培养文件表入库成功，记录数：%d\n", total)

	return nil
}

func (m *UrineCultureService) Merge() (err error) {
	// 循环读取尿液表，收集同一个人一个月内的记录放入待合并对象，收集满则合并或新增并重置对象，若一次循环结束不要合并或重置
	// 下次循环继续判断处理  循环结束后判断对象若没被重置在合并一次
	pageSize := 1000
	pageIndex := 1
	total := 0
	dataToMerge := &DataToMerge{}
	for {
		dataList := make([]*tables.TUrineCulture, 0)
		err = m.db.Table(tables.TableUrineCulture).Select(tables.TableTUrineCultureFields).
			Order("name asc,visitCardId asc,sampleNoTime asc").
			Limit(pageSize).Offset((pageIndex - 1) * pageSize).Find(&dataList).Error
		if nil != err {
			fmt.Printf("获取数据异常 pageIndex：%d\n", pageIndex)
			return err
		}
		count := len(dataList)
		total += count
		if count == 0 {
			break
		}

		for _, dataInfo := range dataList {
			if dataInfo.Name == dataToMerge.name && dataInfo.VisitCardID == dataToMerge.visitCardID {
				// 判断时间是否短期检查，是的话判断写入第几个，保留多次的12和尾，若时间间隔超了则执行合并并重置后录入此条
				if dataInfo.SampleNoTime-dataToMerge.beginTime <= 14 {
					if len(dataToMerge.first) <= 0 {
						fmt.Printf("同一个人第1次 name[%s],id[%s]\n", dataToMerge.name, dataToMerge.visitCardID)
						dataToMerge.first = dataInfo.MicroDataIDName
					} else if len(dataToMerge.second) <= 0 {
						fmt.Printf("同一个人第2次 name[%s],id[%s]\n", dataToMerge.name, dataToMerge.visitCardID)
						dataToMerge.second = dataInfo.MicroDataIDName
					} else {
						fmt.Printf("同一个人第3次 name[%s],id[%s]\n", dataToMerge.name, dataToMerge.visitCardID)
						dataToMerge.third = dataInfo.MicroDataIDName
					}
				} else {
					fmt.Printf("相同人时间超过14天，不同批次检测,合并一下，name[%s],id[%s]\n", dataToMerge.name, dataToMerge.visitCardID)
					// 合并
					_ = m.mateAndWriteCollect(dataToMerge)
					//if err != nil {
					//	continue
					//}
					dataToMerge = &DataToMerge{
						ID:          dataInfo.ID,
						visitCardID: dataInfo.VisitCardID,
						name:        dataInfo.Name,
						age:         dataInfo.Age,
						sex:         dataInfo.Sex,
						diagnosis:   dataInfo.Diagnosis,
						beginTime:   dataInfo.SampleNoTime,
						first:       dataInfo.MicroDataIDName,
					}
				}
			} else { // 换人了 若待和入数据未重置则先合并，然后重新录入此人,若待和入为空直接录入此条继续循环即可
				if len(dataToMerge.name) <= 0 && len(dataToMerge.visitCardID) <= 0 {
					dataToMerge.ID = dataInfo.ID
					dataToMerge.name = dataInfo.Name
					dataToMerge.visitCardID = dataInfo.VisitCardID
					dataToMerge.age = dataInfo.Age
					dataToMerge.sex = dataInfo.Sex
					dataToMerge.diagnosis = dataInfo.Diagnosis
					dataToMerge.beginTime = dataInfo.SampleNoTime
					dataToMerge.first = dataInfo.MicroDataIDName
					fmt.Printf("new to merge name[%s],id[%s]\n", dataToMerge.name, dataToMerge.visitCardID)
					continue
				}
				// 执行合并并重置待合并数据
				fmt.Printf("不同人，合并上一个人数据，name[%s],id[%s]\n", dataToMerge.name, dataToMerge.visitCardID)
				// 合并
				_ = m.mateAndWriteCollect(dataToMerge)
				//if err != nil {
				//	continue
				//}
				dataToMerge = &DataToMerge{
					ID:          dataInfo.ID,
					visitCardID: dataInfo.VisitCardID,
					name:        dataInfo.Name,
					age:         dataInfo.Age,
					sex:         dataInfo.Sex,
					diagnosis:   dataInfo.Diagnosis,
					beginTime:   dataInfo.SampleNoTime,
					first:       dataInfo.MicroDataIDName,
				}
				continue
			}
		}
		pageIndex++
		if count < pageSize {
			//fmt.Printf("QueryTCollectbreak!count:%d,pageSize:%d ", count, pageSize)
			break
		}
	}
	// 若待和入数据还有内容执行合并
	if len(dataToMerge.name) > 0 && len(dataToMerge.visitCardID) > 0 {
		fmt.Printf("最后一条待合并数据处理 name[%s],id[%s]\n", dataToMerge.name, dataToMerge.visitCardID)
		// 合并
		_ = m.mateAndWriteCollect(dataToMerge)
	}
	if total > 0 {
		fmt.Printf("尿液表和入主表完成，匹配不到新增【%d】，匹配成功合并【%d】，匹配冲突新增【%d】，系统异常【%d】\n",
			m.mergeAdd, m.mergeUpdate, m.mergeConflict, m.mergeErr)
	} else {
		fmt.Printf("本次无尿培养数据需要合入\n")
	}

	return nil
}

func (m *UrineCultureService) mateAndWriteCollect(dataToMerge *DataToMerge) error {
	collectList, err := GetCollectList(m.db, dataToMerge.name, dataToMerge.visitCardID,
		dataToMerge.beginTime-30, dataToMerge.beginTime, "asc")
	if err != nil {
		m.mergeErr++
		return err
	}
	if len(collectList) <= 0 {
		m.mergeAdd++
		fmt.Printf("尿液数据找不到可合入的汇总数据,新增数据！！姓名【%s】,就诊卡号[%s]\n", dataToMerge.name,
			dataToMerge.visitCardID)
		collectData := &tables.TCollect{
			Name:        dataToMerge.name,
			VisitCardID: dataToMerge.visitCardID,
			VisitTime:   dataToMerge.beginTime,
			F123:        dataToMerge.first,
			F124:        dataToMerge.second,
			F125:        dataToMerge.third,
			F8:          dataToMerge.age,
			F6:          dataToMerge.sex,
			F13:         dataToMerge.diagnosis,
			//CreateTime:  time.Now().Format("2006-01-02 15:04:05"),
			//UpdateTime:  time.Now().Format("2006-01-02 15:04:05"),
		}
		err := AddCollect(m.db, collectData)
		if err != nil {
			m.mergeErr++
			fmt.Printf("新增尿液数据异常！！姓名【%s】,就诊卡号[%s],err[%s]\n", dataToMerge.name,
				dataToMerge.visitCardID, err.Error())
			return err
		}
		fmt.Printf("新增尿液数据成功！！总表id【%d】,尿液表id【%d】姓名【%s】,就诊卡号[%s] \n",
			collectData.ID, dataToMerge.ID, dataToMerge.name, dataToMerge.visitCardID)
		return nil
	}
	// 匹配冲突则继续 直到最后一条总表记录还冲突的话新增数据
	for i, collectData := range collectList {
		if len(collectData.F123) > 0 || len(collectData.F124) > 0 || len(collectData.F125) > 0 {
			fmt.Printf("尿培养匹配到到总表数据共【%d】条，第【%d】条已存在监测记录，继续匹配！！姓名【%s】,"+
				"就诊卡号[%s] 总表时间【%d】,尿培养表时间【%d】\n", len(collectList), i+1, collectData.Name,
				collectData.VisitCardID, collectData.VisitTime, dataToMerge.beginTime)
			continue
		}
		// 匹配成功 操作合并
		collectData.F123 = dataToMerge.first
		collectData.F124 = dataToMerge.second
		collectData.F125 = dataToMerge.third
		if len(collectData.F8) <= 0 {
			collectData.F8 = dataToMerge.age
		}
		if len(collectData.F6) <= 0 {
			collectData.F6 = dataToMerge.sex
		}
		if len(collectData.F13) <= 0 {
			collectData.F13 = dataToMerge.diagnosis
		}
		err = UpdateCollect(m.db, collectData)
		if err != nil {
			m.mergeErr++
			fmt.Printf("合并尿液数据异常！！姓名【%s】,就诊卡号[%s],err[%s]\n", dataToMerge.name,
				dataToMerge.visitCardID, err.Error())
			return err
		}
		m.mergeUpdate++
		fmt.Printf("合并尿液数据成功！！总表id【%d】,尿液表id【%d】姓名【%s】,就诊卡号[%s] \n",
			collectData.ID, dataToMerge.ID, dataToMerge.name, dataToMerge.visitCardID)
		return nil
	}
	//能走到这的说明所有总表数据都冲突了，新增
	m.mergeConflict++
	fmt.Printf("尿液数据与所有总表数据都冲突,新增数据！！姓名【%s】,就诊卡号[%s]\n", dataToMerge.name,
		dataToMerge.visitCardID)
	collectData := &tables.TCollect{
		Name:        dataToMerge.name,
		VisitCardID: dataToMerge.visitCardID,
		VisitTime:   dataToMerge.beginTime,
		F123:        dataToMerge.first,
		F124:        dataToMerge.second,
		F125:        dataToMerge.third,
		F8:          dataToMerge.age,
		F6:          dataToMerge.sex,
		F13:         dataToMerge.diagnosis,
		//CreateTime:  time.Now().Format("2006-01-02 15:04:05"),
		//UpdateTime:  time.Now().Format("2006-01-02 15:04:05"),
	}
	err = AddCollect(m.db, collectData)
	if err != nil {
		m.mergeErr++
		fmt.Printf("冲突新增尿液数据异常！！姓名【%s】,就诊卡号[%s],err[%s]\n", dataToMerge.name,
			dataToMerge.visitCardID, err.Error())
		return err
	}
	fmt.Printf("冲突新增尿液数据成功！！总表id【%d】,尿液表id【%d】姓名【%s】,就诊卡号[%s] \n",
		collectData.ID, dataToMerge.ID, dataToMerge.name, dataToMerge.visitCardID)
	return nil
}
