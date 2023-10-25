package logic

import (
	"errors"
	"fmt"
	"github.com/pengliu007/forBrotherLee/tables"
	"github.com/tealeg/xlsx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strings"
	"time"
)

// 检验科
type LaboratoryService struct {
	//fileHandle *xlsx.File
	mergeErr            int
	mergeAdd            int
	mergeSuc            int
	mergeConflictAdd    int
	mergeConflictUpdate int
	lastMergeSampleNo   string
	lastMergeSampleTime int
	db                  *gorm.DB
}

func NewLaboratoryService() *LaboratoryService {
	return &LaboratoryService{}
}

func (m *LaboratoryService) InitDb() (err error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/inspectionInfo?charset=utf8mb4&parseTime=false&maxAllowedPacket=104857600"
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	//m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("new db failed,异常 err:%+v", err)
		return err
	}
	sqlStr := "truncate table " + tables.TableLaboratory
	err = m.db.Exec(sqlStr, []interface{}{}...).Error
	if err != nil {
		fmt.Printf("清空检验科科表失败异常")
		return err
	}
	return nil
}

func (m *LaboratoryService) LoadFile(fileName string) (err error) {
	fileHandle, err := xlsx.OpenFile(fileName)
	if err != nil {
		fmt.Printf("检验科表打开失败异常,err:%s", err.Error())
		return err
	}
	fmt.Printf("检验科表打开成功，记录数：%d\n", len(fileHandle.Sheets[0].Rows))

	// 入库
	dataList := make([]*tables.TLaboratory, 0)
	total := 0
	for i, rowInfo := range fileHandle.Sheets[0].Rows {
		if i < 1 { //前1行无需入库
			fmt.Printf("前1行为表头无需入库:%d\n", i)
			continue
		}
		cells := rowInfo.Cells
		if len(cells) < 15 {
			//fmt.Printf("cells [deal_error] 【%s】len[%d]，err\n", cells[13], len(cells))
			continue
		}
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
		dataInfo := &tables.TLaboratory{
			Name:          strings.Trim(cells[5].Value, " "),
			VisitCardID:   strings.Trim(cells[4].Value, " "),
			Age:           strings.Replace(strings.Trim(cells[7].Value, " "), "岁", "", -1),
			Sex:           strings.Trim(cells[6].Value, " "),
			SampleNo:      strings.Trim(cells[1].Value, " "),
			SampleNoTime:  int(excelTime),
			Diagnosis:     strings.Trim(cells[9].Value, " "),
			ProjectName:   strings.Trim(cells[13].Value, " "),
			ProjectResult: strings.Trim(cells[14].Value, " "),
			CreateTime:    time.Now().Format("2006-01-02 15:04:05"),
			UpdateTime:    time.Now().Format("2006-01-02 15:04:05"),
		}
		if len(dataInfo.Name) <= 0 && len(dataInfo.VisitCardID) <= 0 {
			fmt.Printf("姓名和就诊卡号均为空，直接过滤 [deal_error]\n")
			continue
		}
		if len(dataInfo.Name) <= 0 || len(dataInfo.VisitCardID) <= 0 {
			fmt.Printf("姓名：%s 或就诊卡号 :%s,为空 异常\n", dataInfo.Name, dataInfo.VisitCardID)
			//continue
		}
		dataList = append(dataList, dataInfo)
		if i > 2 && i%100 == 0 { // 每100条写入一次 并重置切片
			err := m.db.Table(tables.TableLaboratory).CreateInBatches(dataList, len(dataList)).Error
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
		err := m.db.Table(tables.TableLaboratory).CreateInBatches(dataList, len(dataList)).Error
		if err != nil {
			fmt.Printf("db get 异常err:%s", err.Error())
			return err
		}
		total += len(dataList)
		//fmt.Printf("入库:%d 条,total:%d\n", len(collectList), total)
		dataList = dataList[0:0]
	}
	fmt.Printf("检验科表入库成功，记录数：%d\n", total)
	return nil
}

func (m *LaboratoryService) Merge() (err error) {
	// 循环读取检验表，找总表4天内的记录合并，若遇到冲突则新增记录不做覆盖（当天的冲突覆盖）
	total := 0
	pageSize := 1000
	pageIndex := 1
	for {
		if total > 0 && total%20000 == 0 {
			fmt.Printf("==========================total:%d\n", total)
		}
		dataList := make([]*tables.TLaboratory, 0)
		err = m.db.Table(tables.TableLaboratory).Select(tables.TableLaboratoryFields).
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
		// 逐条匹配总表，找到写入，没找到过滤,若找到的记录列已存在数据则暂时打印日志不做覆盖
		for _, dataInfo := range dataList {
			err = m.mateAndWriteCollect(dataInfo)
			if err != nil {
				continue
			}
		}
		pageIndex++
		if count < pageSize {
			//fmt.Printf("QueryTCollectbreak!count:%d,pageSize:%d ", count, pageSize)
			break
		}
	}
	if total > 0 {
		fmt.Printf("检验包合入主表完成，匹配成功合并【%d】，匹配不到新增【%d】，匹配冲突新增【%d】,匹配冲突覆盖【%d】，"+
			"系统异常【%d】\n", m.mergeSuc, m.mergeAdd, m.mergeConflictAdd, m.mergeConflictUpdate, m.mergeErr)
	} else {
		fmt.Printf("本次无检验数据需要合入\n")
	}
	return nil
}

func (m *LaboratoryService) mateAndWriteCollect(dataLaboratory *tables.TLaboratory) (err error) {
	_, ok := projectNameMap[dataLaboratory.ProjectName]
	if !ok {
		fmt.Printf("检验项目映射主表找不到对应列！！姓名【%s】,就诊卡号[%s],检验项目[%s]\n", dataLaboratory.Name,
			dataLaboratory.VisitCardID, dataLaboratory.ProjectName)
		return errors.New("检验项目映射主表找不到对应列异常 ")
	}
	// 取最近的一次 所以desc
	collectList, err := GetCollectList(m.db, dataLaboratory.Name, dataLaboratory.VisitCardID,
		dataLaboratory.SampleNoTime-4, dataLaboratory.SampleNoTime, "desc")
	if err != nil {
		m.mergeErr++
		return err
	}
	if len(collectList) <= 0 {
		collectInfo := m.getMergerCollectInfo(nil, dataLaboratory)
		if collectInfo == nil {
			fmt.Printf("检验数据匹配不到映射表，过滤.姓名【%s】,就诊卡号[%s],项目【%s】\n", dataLaboratory.Name,
				dataLaboratory.VisitCardID, dataLaboratory.ProjectName)
			return nil
		}
		err = AddCollect(m.db, collectInfo)
		if err != nil {
			m.mergeErr++
			fmt.Printf("检验数据 merge 异常！！姓名【%s】,就诊卡号[%s],新增汇总数据失败：%s\n", dataLaboratory.Name,
				dataLaboratory.VisitCardID, err.Error())
			return err
		}
		m.mergeAdd++
		fmt.Printf("检验数据找不到可合入的汇总数据！！姓名【%s】,就诊卡号[%s],新增汇总数据成功\n", dataLaboratory.Name,
			dataLaboratory.VisitCardID)
		return nil
	}
	// 这样丢弃不行，一次检查可能多管血 对应多个标本
	//if dataLaboratory.SampleNo != m.lastMergeSampleNo && dataLaboratory.SampleNoTime-m.lastMergeSampleTime < 4 {
	//	// 判断上次和入记录时间，4天内的检验丢弃只和入第一次
	//	fmt.Printf("4天内检验数据丢弃！！姓名【%s】,就诊卡号[%s]，标本号[%s],标本时间[%d],\n", dataLaboratory.Name,
	//		dataLaboratory.VisitCardID, dataLaboratory.SampleNo, dataLaboratory.SampleNoTime)
	//	return nil
	//}
	// 检查是否冲突，是的话判断是否同一天检查，若是同一天检查的冲突则覆盖，否则新增。无冲突直接合并
	m.lastMergeSampleTime = dataLaboratory.SampleNoTime
	m.lastMergeSampleNo = dataLaboratory.SampleNo
	// 判断是否冲突  冲突新增，无冲突则合并
	isConflict := m.checkMergerConflict(collectList[0], dataLaboratory)
	if !isConflict {
		collectInfo := m.getMergerCollectInfo(collectList[0], dataLaboratory)
		if collectInfo == nil {
			fmt.Printf("检验数据匹配不到映射表，过滤.姓名【%s】,就诊卡号[%s],项目【%s】\n", dataLaboratory.Name,
				dataLaboratory.VisitCardID, dataLaboratory.ProjectName)
			return nil
		}
		err = UpdateCollect(m.db, collectInfo)
		if err != nil {
			m.mergeErr++
			fmt.Printf("检验数据合并汇总数据异常！！姓名【%s】,就诊卡号[%s]err：%s\n", dataLaboratory.Name,
				dataLaboratory.VisitCardID, err.Error())
			return err
		}
		m.mergeSuc++
		fmt.Printf("检验数据正常合入汇总数据成功！！姓名【%s】,就诊卡号[%s]\n", dataLaboratory.Name,
			dataLaboratory.VisitCardID)
	} else {
		if collectList[0].VisitTime != dataLaboratory.SampleNoTime {
			fmt.Printf("检验数据合并汇总数据冲突,时间不同新增！！姓名【%s】,就诊卡号[%s]\n", dataLaboratory.Name,
				dataLaboratory.VisitCardID)
			collectInfo := m.getMergerCollectInfo(nil, dataLaboratory)
			if collectInfo == nil {
				fmt.Printf("检验数据匹配不到映射表，过滤.姓名【%s】,就诊卡号[%s],项目【%s】\n", dataLaboratory.Name,
					dataLaboratory.VisitCardID, dataLaboratory.ProjectName)
				return nil
			}
			err = AddCollect(m.db, collectInfo)
			if err != nil {
				m.mergeErr++
				fmt.Printf("检验数据合并汇总数据冲突新增异常！！姓名【%s】,就诊卡号[%s]err：%s\n", dataLaboratory.Name,
					dataLaboratory.VisitCardID, err.Error())
				return err
			}
			m.mergeConflictAdd++
		} else {
			fmt.Printf("检验数据合并汇总数据冲突,时间相同覆盖！！姓名【%s】,就诊卡号[%s]\n", dataLaboratory.Name,
				dataLaboratory.VisitCardID)
			collectInfo := m.getMergerCollectInfo(collectList[0], dataLaboratory)
			if collectInfo == nil {
				fmt.Printf("检验数据匹配不到映射表，过滤.姓名【%s】,就诊卡号[%s],项目【%s】\n", dataLaboratory.Name,
					dataLaboratory.VisitCardID, dataLaboratory.ProjectName)
				return nil
			}
			err = UpdateCollect(m.db, collectInfo)
			if err != nil {
				m.mergeErr++
				fmt.Printf("检验数据合并汇总数据冲突覆盖异常！！姓名【%s】,就诊卡号[%s]err：%s\n", dataLaboratory.Name,
					dataLaboratory.VisitCardID, err.Error())
				return err
			}
			m.mergeConflictUpdate++
		}

		fmt.Printf("检验数据合入汇总数据冲突新增成功！！姓名【%s】,就诊卡号[%s],新增汇总数据成功\n", dataLaboratory.Name,
			dataLaboratory.VisitCardID)
	}

	return nil
}
func (m *LaboratoryService) checkMergerConflict(collectInfo *tables.TCollect, dataLaboratory *tables.TLaboratory) (isConflict bool) {
	isConflict = false
	// 外层已判断过mapkey 直接用即可
	projectName := projectNameMap[dataLaboratory.ProjectName]
	if strings.Trim(projectName, " ") == "甲胎蛋白" && len(collectInfo.F99) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "糖链抗原CA19-9" && len(collectInfo.F100) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "糖链抗原CA125" && len(collectInfo.F101) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "癌胚抗原(CEA)" && len(collectInfo.F102) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "糖类抗原CA15-3" && len(collectInfo.F103) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "糖链抗原CA72-4" && len(collectInfo.F104) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗Jo-1" && len(collectInfo.F154) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "天门冬氨酸转氨酶(AST)" && len(collectInfo.F39) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "线粒体型天门冬氨酸转氨酶" && len(collectInfo.F37) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "维生素D3" && len(collectInfo.F105) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "总三碘甲状腺原氨酸" && len(collectInfo.F112) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "促甲状腺激素" && len(collectInfo.F114) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "ANA（IIF）" && len(collectInfo.F162) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体（定性）" && len(collectInfo.F144) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗PML" && len(collectInfo.F169) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗线粒体M2" && len(collectInfo.F174) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "pANCA" && len(collectInfo.F175) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "cANCA" && len(collectInfo.F176) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核小体" && len(collectInfo.F158) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗sm抗体" && len(collectInfo.F177) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗SSA" && len(collectInfo.F149) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗SSB" && len(collectInfo.F151) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗ScL-70" && len(collectInfo.F152) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗PM-Scl" && len(collectInfo.F153) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗Ro-52" && len(collectInfo.F150) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体（1：10）" && len(collectInfo.F126) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体（1：20）" && len(collectInfo.F127) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体(1:100)" && len(collectInfo.F131) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体(1:160)" && len(collectInfo.F132) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体(1:320)" && len(collectInfo.F133) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体(1:10000)" && len(collectInfo.F140) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "1：100000" && len(collectInfo.F142) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体(1:1000)" && len(collectInfo.F135) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体(1:2560)" && len(collectInfo.F137) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体(1:1280)" && len(collectInfo.F136) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体(1:3200)" && len(collectInfo.F138) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗nRNP/Sm" && len(collectInfo.F147) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核抗体(1:32)" && len(collectInfo.F128) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "快速血沉试验" && len(collectInfo.F109) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "血清免疫球蛋白M" && len(collectInfo.F82) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "血清免疫球蛋白A" && len(collectInfo.F81) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "血清免疫球蛋白G" && len(collectInfo.F80) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "超敏C反应蛋白" && len(collectInfo.F88) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿白细胞定量" && len(collectInfo.F121) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗EBV衣壳抗体IgG" && len(collectInfo.F90) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗EBV早期抗体IgG" && len(collectInfo.F92) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗着丝点蛋白B" && len(collectInfo.F155) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "1,3-β-D葡聚糖(血液)" && len(collectInfo.F107) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "胱抑素C" && len(collectInfo.F68) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "免疫球蛋白G亚类4" && len(collectInfo.F84) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "免疫球蛋白G亚类3" && len(collectInfo.F85) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "免疫球蛋白G亚类2" && len(collectInfo.F86) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "免疫球蛋白G亚类1" && len(collectInfo.F87) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "白细胞计数" && len(collectInfo.F27) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "中性粒细胞绝对值" && len(collectInfo.F28) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "淋巴细胞绝对值" && len(collectInfo.F29) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "单核细胞绝对值" && len(collectInfo.F30) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "红细胞计数" && len(collectInfo.F31) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "血红蛋白" && len(collectInfo.F32) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "血细胞比容" && len(collectInfo.F33) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "平均红细胞体积" && len(collectInfo.F34) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "血小板计数" && len(collectInfo.F35) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "腺苷脱氨酶" && len(collectInfo.F36) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "丙氨酸氨基转移酶(ALT)" && len(collectInfo.F38) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "总蛋白" && len(collectInfo.F40) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "球蛋白" && len(collectInfo.F41) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "白蛋白" && len(collectInfo.F42) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "总胆红素" && len(collectInfo.F43) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "直接胆红素" && len(collectInfo.F44) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "间接胆红素" && len(collectInfo.F45) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "碱性磷酸酶" && len(collectInfo.F46) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "γ-谷氨酰基转移酶" && len(collectInfo.F47) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "总胆汁酸" && len(collectInfo.F48) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "亮氨酸氨基肽酶" && len(collectInfo.F49) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "白蛋白/球蛋白" && len(collectInfo.F50) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "AST/ALT" && len(collectInfo.F51) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "总胆固醇" && len(collectInfo.F52) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "甘油三脂" && len(collectInfo.F53) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "高密度脂蛋白胆固醇" && len(collectInfo.F54) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "低密度脂蛋白胆固醇(LDL_C)" && len(collectInfo.F55) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿素" && len(collectInfo.F56) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "肌酐" && len(collectInfo.F57) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "葡萄糖" && len(collectInfo.F58) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "糖化血红蛋白" && len(collectInfo.F59) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿酸" && len(collectInfo.F60) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "载脂蛋白A1" && len(collectInfo.F61) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "载脂蛋白B" && len(collectInfo.F62) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "钾" && len(collectInfo.F63) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "钠" && len(collectInfo.F64) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "氯" && len(collectInfo.F65) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "总钙" && len(collectInfo.F66) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "二氧化碳" && len(collectInfo.F67) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "无机磷" && len(collectInfo.F69) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "肌酸激酶" && len(collectInfo.F70) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "血清胆碱脂酶" && len(collectInfo.F71) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "凝血酶原时间" && len(collectInfo.F72) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "活化部分凝血活酶时间" && len(collectInfo.F73) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "纤维蛋白原含量" && len(collectInfo.F74) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "凝血酶时间" && len(collectInfo.F75) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "D-二聚体" && len(collectInfo.F76) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "纤维蛋白原降解产物(血浆)" && len(collectInfo.F77) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "凝血酶原活动度" && len(collectInfo.F78) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "PT国际标准化比值" && len(collectInfo.F79) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "血清免疫球蛋白E" && len(collectInfo.F83) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "巨细胞病毒抗体IgM" && len(collectInfo.F89) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗EBV核抗体IgG" && len(collectInfo.F91) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗EBV衣壳抗体IgM" && len(collectInfo.F93) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗EBV壳抗体IgG亲合力" && len(collectInfo.F94) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "EBV Zta蛋白抗体IgA" && len(collectInfo.F95) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗EBV衣壳抗体IgA" && len(collectInfo.F96) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗EBV核抗体IgA" && len(collectInfo.F97) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "异常凝血酶原" && len(collectInfo.F98) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "内毒素定量(血液)" && len(collectInfo.F106) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "血浆氨" && len(collectInfo.F108) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "游离三碘甲状腺原氨酸" && len(collectInfo.F110) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "游离甲状腺素" && len(collectInfo.F111) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "总甲状腺素" && len(collectInfo.F113) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿白细胞定性" && len(collectInfo.F115) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿蛋白定性" && len(collectInfo.F116) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿胆原定性" && len(collectInfo.F117) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿胆红素定性" && len(collectInfo.F118) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿红细胞定性" && len(collectInfo.F119) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿红细胞定量" && len(collectInfo.F120) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "尿细菌定量" && len(collectInfo.F122) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗U1-nRNP抗体" && len(collectInfo.F145) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗U1-snRNP抗体" && len(collectInfo.F146) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗增殖细胞核抗原" && len(collectInfo.F156) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗双链DNA抗体" && len(collectInfo.F157) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗组蛋白" && len(collectInfo.F159) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗核糖体P蛋白" && len(collectInfo.F160) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "自身免疫性肝病抗体检测" && len(collectInfo.F161) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗线粒体抗体" && len(collectInfo.F163) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗肝肾微粒体" && len(collectInfo.F164) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗肝抗原" && len(collectInfo.F165) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗平滑肌抗体" && len(collectInfo.F166) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗3E（BPO）" && len(collectInfo.F167) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗Sp100" && len(collectInfo.F168) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗gp210" && len(collectInfo.F170) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗肝肾微粒体抗体" && len(collectInfo.F171) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗肝细胞溶质抗原I抗体" && len(collectInfo.F172) > 0 {
		isConflict = true
	} else if strings.Trim(projectName, " ") == "抗可溶性肝抗原/肝胰抗原抗体" && len(collectInfo.F173) > 0 {
		isConflict = true
	}
	return isConflict
}

func (m *LaboratoryService) getMergerCollectInfo(collectInfo *tables.TCollect, dataLaboratory *tables.TLaboratory) *tables.TCollect {
	// 外层已判断过mapkey 直接用即可
	var mergeCollectInfo *tables.TCollect
	if collectInfo == nil {
		mergeCollectInfo = &tables.TCollect{
			Name:        dataLaboratory.Name,
			VisitCardID: dataLaboratory.VisitCardID,
			VisitTime:   dataLaboratory.SampleNoTime,
		}
	} else {
		mergeCollectInfo = collectInfo
	}
	mergeCollectInfo.F8 = dataLaboratory.Age
	mergeCollectInfo.F6 = dataLaboratory.Sex
	mergeCollectInfo.F13 = dataLaboratory.Diagnosis
	projectName := projectNameMap[dataLaboratory.ProjectName]
	if projectName == "甲胎蛋白" {
		mergeCollectInfo.F99 = dataLaboratory.ProjectResult
	} else if projectName == "糖链抗原CA19-9" {
		mergeCollectInfo.F100 = dataLaboratory.ProjectResult
	} else if projectName == "糖链抗原CA125" {
		mergeCollectInfo.F101 = dataLaboratory.ProjectResult
	} else if projectName == "癌胚抗原(CEA)" {
		mergeCollectInfo.F102 = dataLaboratory.ProjectResult
	} else if projectName == "糖类抗原CA15-3" {
		mergeCollectInfo.F103 = dataLaboratory.ProjectResult
	} else if projectName == "糖链抗原CA72-4" {
		mergeCollectInfo.F104 = dataLaboratory.ProjectResult
	} else if projectName == "抗Jo-1" {
		mergeCollectInfo.F154 = dataLaboratory.ProjectResult
	} else if projectName == "天门冬氨酸转氨酶(AST)" {
		mergeCollectInfo.F39 = dataLaboratory.ProjectResult
	} else if projectName == "线粒体型天门冬氨酸转氨酶" {
		mergeCollectInfo.F37 = dataLaboratory.ProjectResult
	} else if projectName == "维生素D3" {
		mergeCollectInfo.F105 = dataLaboratory.ProjectResult
	} else if projectName == "总三碘甲状腺原氨酸" {
		mergeCollectInfo.F112 = dataLaboratory.ProjectResult
	} else if projectName == "促甲状腺激素" {
		mergeCollectInfo.F114 = dataLaboratory.ProjectResult
	} else if projectName == "ANA（IIF）" {
		mergeCollectInfo.F162 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体（定性）" {
		mergeCollectInfo.F144 = dataLaboratory.ProjectResult
	} else if projectName == "抗PML" {
		mergeCollectInfo.F169 = dataLaboratory.ProjectResult
	} else if projectName == "抗线粒体M2" {
		mergeCollectInfo.F174 = dataLaboratory.ProjectResult
	} else if projectName == "pANCA" {
		mergeCollectInfo.F175 = dataLaboratory.ProjectResult
	} else if projectName == "cANCA" {
		mergeCollectInfo.F176 = dataLaboratory.ProjectResult
	} else if projectName == "抗核小体" {
		mergeCollectInfo.F158 = dataLaboratory.ProjectResult
	} else if projectName == "抗sm抗体" {
		mergeCollectInfo.F177 = dataLaboratory.ProjectResult
	} else if projectName == "抗SSA" {
		mergeCollectInfo.F149 = dataLaboratory.ProjectResult
	} else if projectName == "抗SSB" {
		mergeCollectInfo.F151 = dataLaboratory.ProjectResult
	} else if projectName == "抗ScL-70" {
		mergeCollectInfo.F152 = dataLaboratory.ProjectResult
	} else if projectName == "抗PM-Scl" {
		mergeCollectInfo.F153 = dataLaboratory.ProjectResult
	} else if projectName == "抗Ro-52" {
		mergeCollectInfo.F150 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体（1：10）" {
		mergeCollectInfo.F126 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体（1：20）" {
		mergeCollectInfo.F127 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体(1:100)" {
		mergeCollectInfo.F131 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体(1:160)" {
		mergeCollectInfo.F132 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体(1:320)" {
		mergeCollectInfo.F133 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体(1:10000)" {
		mergeCollectInfo.F140 = dataLaboratory.ProjectResult
	} else if projectName == "1：100000" {
		mergeCollectInfo.F142 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体(1:1000)" {
		mergeCollectInfo.F135 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体(1:2560)" {
		mergeCollectInfo.F137 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体(1:1280)" {
		mergeCollectInfo.F136 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体(1:3200)" {
		mergeCollectInfo.F138 = dataLaboratory.ProjectResult
	} else if projectName == "抗nRNP/Sm" {
		mergeCollectInfo.F147 = dataLaboratory.ProjectResult
	} else if projectName == "抗核抗体(1:32)" {
		mergeCollectInfo.F128 = dataLaboratory.ProjectResult
	} else if projectName == "快速血沉试验" {
		mergeCollectInfo.F109 = dataLaboratory.ProjectResult
	} else if projectName == "血清免疫球蛋白M" {
		mergeCollectInfo.F82 = dataLaboratory.ProjectResult
	} else if projectName == "血清免疫球蛋白A" {
		mergeCollectInfo.F81 = dataLaboratory.ProjectResult
	} else if projectName == "血清免疫球蛋白G" {
		mergeCollectInfo.F80 = dataLaboratory.ProjectResult
	} else if projectName == "超敏C反应蛋白" {
		mergeCollectInfo.F88 = dataLaboratory.ProjectResult
	} else if projectName == "尿白细胞定量" {
		mergeCollectInfo.F121 = dataLaboratory.ProjectResult
	} else if projectName == "抗EBV衣壳抗体IgG" {
		mergeCollectInfo.F90 = dataLaboratory.ProjectResult
	} else if projectName == "抗EBV早期抗体IgG" {
		mergeCollectInfo.F92 = dataLaboratory.ProjectResult
	} else if projectName == "抗着丝点蛋白B" {
		mergeCollectInfo.F155 = dataLaboratory.ProjectResult
	} else if projectName == "1,3-β-D葡聚糖(血液)" {
		mergeCollectInfo.F107 = dataLaboratory.ProjectResult
	} else if projectName == "胱抑素C" {
		mergeCollectInfo.F68 = dataLaboratory.ProjectResult
	} else if projectName == "免疫球蛋白G亚类4" {
		mergeCollectInfo.F84 = dataLaboratory.ProjectResult
	} else if projectName == "免疫球蛋白G亚类3" {
		mergeCollectInfo.F85 = dataLaboratory.ProjectResult
	} else if projectName == "免疫球蛋白G亚类2" {
		mergeCollectInfo.F86 = dataLaboratory.ProjectResult
	} else if projectName == "免疫球蛋白G亚类1" {
		mergeCollectInfo.F87 = dataLaboratory.ProjectResult
	} else if projectName == "白细胞计数" {
		mergeCollectInfo.F27 = dataLaboratory.ProjectResult
	} else if projectName == "中性粒细胞绝对值" {
		mergeCollectInfo.F28 = dataLaboratory.ProjectResult
	} else if projectName == "淋巴细胞绝对值" {
		mergeCollectInfo.F29 = dataLaboratory.ProjectResult
	} else if projectName == "单核细胞绝对值" {
		mergeCollectInfo.F30 = dataLaboratory.ProjectResult
	} else if projectName == "红细胞计数" {
		mergeCollectInfo.F31 = dataLaboratory.ProjectResult
	} else if projectName == "血红蛋白" {
		mergeCollectInfo.F32 = dataLaboratory.ProjectResult
	} else if projectName == "血细胞比容" {
		mergeCollectInfo.F33 = dataLaboratory.ProjectResult
	} else if projectName == "平均红细胞体积" {
		mergeCollectInfo.F34 = dataLaboratory.ProjectResult
	} else if projectName == "血小板计数" {
		mergeCollectInfo.F35 = dataLaboratory.ProjectResult
	} else if projectName == "腺苷脱氨酶" {
		mergeCollectInfo.F36 = dataLaboratory.ProjectResult
	} else if projectName == "丙氨酸氨基转移酶(ALT)" {
		mergeCollectInfo.F38 = dataLaboratory.ProjectResult
	} else if projectName == "总蛋白" {
		mergeCollectInfo.F40 = dataLaboratory.ProjectResult
	} else if projectName == "球蛋白" {
		mergeCollectInfo.F41 = dataLaboratory.ProjectResult
	} else if projectName == "白蛋白" {
		mergeCollectInfo.F42 = dataLaboratory.ProjectResult
	} else if projectName == "总胆红素" {
		mergeCollectInfo.F43 = dataLaboratory.ProjectResult
	} else if projectName == "直接胆红素" {
		mergeCollectInfo.F44 = dataLaboratory.ProjectResult
	} else if projectName == "间接胆红素" {
		mergeCollectInfo.F45 = dataLaboratory.ProjectResult
	} else if projectName == "碱性磷酸酶" {
		mergeCollectInfo.F46 = dataLaboratory.ProjectResult
	} else if projectName == "γ-谷氨酰基转移酶" {
		mergeCollectInfo.F47 = dataLaboratory.ProjectResult
	} else if projectName == "总胆汁酸" {
		mergeCollectInfo.F48 = dataLaboratory.ProjectResult
	} else if projectName == "亮氨酸氨基肽酶" {
		mergeCollectInfo.F49 = dataLaboratory.ProjectResult
	} else if projectName == "白蛋白/球蛋白" {
		mergeCollectInfo.F50 = dataLaboratory.ProjectResult
	} else if projectName == "AST/ALT" {
		mergeCollectInfo.F51 = dataLaboratory.ProjectResult
	} else if projectName == "总胆固醇" {
		mergeCollectInfo.F52 = dataLaboratory.ProjectResult
	} else if projectName == "甘油三脂" {
		mergeCollectInfo.F53 = dataLaboratory.ProjectResult
	} else if projectName == "高密度脂蛋白胆固醇" {
		mergeCollectInfo.F54 = dataLaboratory.ProjectResult
	} else if projectName == "低密度脂蛋白胆固醇(LDL_C)" {
		mergeCollectInfo.F55 = dataLaboratory.ProjectResult
	} else if projectName == "尿素" {
		mergeCollectInfo.F56 = dataLaboratory.ProjectResult
	} else if projectName == "肌酐" {
		mergeCollectInfo.F57 = dataLaboratory.ProjectResult
	} else if projectName == "葡萄糖" {
		mergeCollectInfo.F58 = dataLaboratory.ProjectResult
	} else if projectName == "糖化血红蛋白" {
		mergeCollectInfo.F59 = dataLaboratory.ProjectResult
	} else if projectName == "尿酸" {
		mergeCollectInfo.F60 = dataLaboratory.ProjectResult
	} else if projectName == "载脂蛋白A1" {
		mergeCollectInfo.F61 = dataLaboratory.ProjectResult
	} else if projectName == "载脂蛋白B" {
		mergeCollectInfo.F62 = dataLaboratory.ProjectResult
	} else if projectName == "钾" {
		mergeCollectInfo.F63 = dataLaboratory.ProjectResult
	} else if projectName == "钠" {
		mergeCollectInfo.F64 = dataLaboratory.ProjectResult
	} else if projectName == "氯" {
		mergeCollectInfo.F65 = dataLaboratory.ProjectResult
	} else if projectName == "总钙" {
		mergeCollectInfo.F66 = dataLaboratory.ProjectResult
	} else if projectName == "二氧化碳" {
		mergeCollectInfo.F67 = dataLaboratory.ProjectResult
	} else if projectName == "无机磷" {
		mergeCollectInfo.F69 = dataLaboratory.ProjectResult
	} else if projectName == "肌酸激酶" {
		mergeCollectInfo.F70 = dataLaboratory.ProjectResult
	} else if projectName == "血清胆碱脂酶" {
		mergeCollectInfo.F71 = dataLaboratory.ProjectResult
	} else if projectName == "凝血酶原时间" {
		mergeCollectInfo.F72 = dataLaboratory.ProjectResult
	} else if projectName == "活化部分凝血活酶时间" {
		mergeCollectInfo.F73 = dataLaboratory.ProjectResult
	} else if projectName == "纤维蛋白原含量" {
		mergeCollectInfo.F74 = dataLaboratory.ProjectResult
	} else if projectName == "凝血酶时间" {
		mergeCollectInfo.F75 = dataLaboratory.ProjectResult
	} else if projectName == "D-二聚体" {
		mergeCollectInfo.F76 = dataLaboratory.ProjectResult
	} else if projectName == "纤维蛋白原降解产物(血浆)" {
		mergeCollectInfo.F77 = dataLaboratory.ProjectResult
	} else if projectName == "凝血酶原活动度" {
		mergeCollectInfo.F78 = dataLaboratory.ProjectResult
	} else if projectName == "PT国际标准化比值" {
		mergeCollectInfo.F79 = dataLaboratory.ProjectResult
	} else if projectName == "血清免疫球蛋白E" {
		mergeCollectInfo.F83 = dataLaboratory.ProjectResult
	} else if projectName == "巨细胞病毒抗体IgM" {
		mergeCollectInfo.F89 = dataLaboratory.ProjectResult
	} else if projectName == "抗EBV核抗体IgG" {
		mergeCollectInfo.F91 = dataLaboratory.ProjectResult
	} else if projectName == "抗EBV衣壳抗体IgM" {
		mergeCollectInfo.F93 = dataLaboratory.ProjectResult
	} else if projectName == "抗EBV壳抗体IgG亲合力" {
		mergeCollectInfo.F94 = dataLaboratory.ProjectResult
	} else if projectName == "EBV Zta蛋白抗体IgA" {
		mergeCollectInfo.F95 = dataLaboratory.ProjectResult
	} else if projectName == "抗EBV衣壳抗体IgA" {
		mergeCollectInfo.F96 = dataLaboratory.ProjectResult
	} else if projectName == "抗EBV核抗体IgA" {
		mergeCollectInfo.F97 = dataLaboratory.ProjectResult
	} else if projectName == "异常凝血酶原" {
		mergeCollectInfo.F98 = dataLaboratory.ProjectResult
	} else if projectName == "内毒素定量(血液)" {
		mergeCollectInfo.F106 = dataLaboratory.ProjectResult
	} else if projectName == "血浆氨" {
		mergeCollectInfo.F108 = dataLaboratory.ProjectResult
	} else if projectName == "游离三碘甲状腺原氨酸" {
		mergeCollectInfo.F110 = dataLaboratory.ProjectResult
	} else if projectName == "游离甲状腺素" {
		mergeCollectInfo.F111 = dataLaboratory.ProjectResult
	} else if projectName == "总甲状腺素" {
		mergeCollectInfo.F113 = dataLaboratory.ProjectResult
	} else if projectName == "尿白细胞定性" {
		mergeCollectInfo.F115 = dataLaboratory.ProjectResult
	} else if projectName == "尿蛋白定性" {
		mergeCollectInfo.F116 = dataLaboratory.ProjectResult
	} else if projectName == "尿胆原定性" {
		mergeCollectInfo.F117 = dataLaboratory.ProjectResult
	} else if projectName == "尿胆红素定性" {
		mergeCollectInfo.F118 = dataLaboratory.ProjectResult
	} else if projectName == "尿红细胞定性" {
		mergeCollectInfo.F119 = dataLaboratory.ProjectResult
	} else if projectName == "尿红细胞定量" {
		mergeCollectInfo.F120 = dataLaboratory.ProjectResult
	} else if projectName == "尿细菌定量" {
		mergeCollectInfo.F122 = dataLaboratory.ProjectResult
	} else if projectName == "抗U1-nRNP抗体" {
		mergeCollectInfo.F145 = dataLaboratory.ProjectResult
	} else if projectName == "抗U1-snRNP抗体" {
		mergeCollectInfo.F146 = dataLaboratory.ProjectResult
	} else if projectName == "抗增殖细胞核抗原" {
		mergeCollectInfo.F156 = dataLaboratory.ProjectResult
	} else if projectName == "抗双链DNA抗体" {
		mergeCollectInfo.F157 = dataLaboratory.ProjectResult
	} else if projectName == "抗组蛋白" {
		mergeCollectInfo.F159 = dataLaboratory.ProjectResult
	} else if projectName == "抗核糖体P蛋白" {
		mergeCollectInfo.F160 = dataLaboratory.ProjectResult
	} else if projectName == "自身免疫性肝病抗体检测" {
		mergeCollectInfo.F161 = dataLaboratory.ProjectResult
	} else if projectName == "抗线粒体抗体" {
		mergeCollectInfo.F163 = dataLaboratory.ProjectResult
	} else if projectName == "抗肝肾微粒体" {
		mergeCollectInfo.F164 = dataLaboratory.ProjectResult
	} else if projectName == "抗肝抗原" {
		mergeCollectInfo.F165 = dataLaboratory.ProjectResult
	} else if projectName == "抗平滑肌抗体" {
		mergeCollectInfo.F166 = dataLaboratory.ProjectResult
	} else if projectName == "抗3E（BPO）" {
		mergeCollectInfo.F167 = dataLaboratory.ProjectResult
	} else if projectName == "抗Sp100" {
		mergeCollectInfo.F168 = dataLaboratory.ProjectResult
	} else if projectName == "抗gp210" {
		mergeCollectInfo.F170 = dataLaboratory.ProjectResult
	} else if projectName == "抗肝肾微粒体抗体" {
		mergeCollectInfo.F171 = dataLaboratory.ProjectResult
	} else if projectName == "抗肝细胞溶质抗原I抗体" {
		mergeCollectInfo.F172 = dataLaboratory.ProjectResult
	} else if projectName == "抗可溶性肝抗原/肝胰抗原抗体" {
		mergeCollectInfo.F173 = dataLaboratory.ProjectResult
	} else {
		return nil
	}

	return mergeCollectInfo
}
