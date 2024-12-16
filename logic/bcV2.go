package logic

import (
	"fmt"
	"github.com/pengliu007/forBrotherLee/tables"
	"github.com/tealeg/xlsx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// B超
type BcV2Service struct {
	//fileHandle *xlsx.File
	mergeErr      int
	mergeSuc      int
	mergeConflict int
	mergeFaild    int
	db            *gorm.DB
}

const (
	F193Key1 = " 肝脏 "
	F193Key2 = "  肝 脏："
	F193Key3 = "肝 脏："
	F193Key4 = "肝脏  "
	F194Key1 = " 胆囊 "
	F194Key2 = "  胆 囊："
	F195Key1 = " 胰腺 "
	F195Key2 = "  胰 腺："
	F196Key1 = " 脾脏 "
	F196Key2 = "  脾 脏："
	F197Key1 = " 门静脉 "
	F197Key2 = "  门静脉"
	F198Key1 = " 脾静脉 "
	F198Key2 = "  脾静脉"
	F199Key1 = " 腹腔 "
	F199Key2 = "腹 腔："
	F200Key1 = " 右侧颈总动脉内-中膜"
	F200Key2 = " 右侧颈总动脉内中膜"
	F247Key1 = " 双肾 "
	F247Key2 = "双肾  "
	F247Key3 = " 双肾："
	F248Key1 = " 甲状腺 "
	F248Key2 = "    甲状腺"
	F248Key3 = "甲状腺  "
	F249Key1 = " 颈部 "
	F249Key2 = "  双侧颈部"
	F249Key3 = "  颈 部："
)

func NewBcV2Service() *BcV2Service {
	return &BcV2Service{}
}

func (m *BcV2Service) InitDb() (err error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/inspectionInfo?charset=utf8mb4&parseTime=false&maxAllowedPacket=104857600"
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	//m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("new db 异常 failed, err:%+v", err)
		return err
	}

	sqlStr := "truncate table " + tables.TableBcV2
	err = m.db.Exec(sqlStr, []interface{}{}...).Error
	if err != nil {
		fmt.Printf("清空消化科B超表失败异常")
		return err
	}
	return nil
}

func (m *BcV2Service) LoadFile(fileName string) (err error) {
	fileHandle, err := xlsx.OpenFile(fileName)
	if err != nil {
		fmt.Printf("B超表打开失败异常,err:%s", err.Error())
		return err
	}
	fmt.Printf("B超表打开成功，记录数：%d\n", len(fileHandle.Sheets[0].Rows))

	// 入库
	dataList := make([]*tables.TBcV2, 0)
	total := 0
	for i, rowInfo := range fileHandle.Sheets[0].Rows {
		if i < 1 { //前1行无需入库
			fmt.Printf("首行为表头无需入库:%d\n", i)
			continue
		}
		cells := rowInfo.Cells
		if len(cells) < 16 {
			fmt.Printf("cells [异常] len[%d]，err\n", len(cells))
			continue
		}

		timeSrc := strings.Trim(cells[8].Value, " ")
		var visitTime float64
		if len(timeSrc) <= 0 {
			fmt.Printf("B超表表记录就诊时间异常,忽律此条：姓名:%s，卡号:%s,时间【%s】\n",
				cells[1].Value, cells[0].Value, cells[8].Value)
			continue
		}
		visitTime, err = strconv.ParseFloat(timeSrc, 64)
		if err != nil {
			fmt.Printf("B超表表记录就诊时间异常2：姓名:%s，卡号:%s,时间【%s】\n",
				cells[1].Value, cells[0].Value, cells[8].Value)
			return err
		}

		dataInfo := &tables.TBcV2{
			Name:         strings.Trim(cells[1].Value, " "),
			VisitCardID:  strings.Trim(cells[0].Value, " "),
			VisitTime:    int(visitTime),
			Age:          strings.Replace(strings.Trim(cells[3].Value, " "), "岁", "", -1),
			Sex:          strings.Trim(cells[2].Value, " "),
			CheckResult:  strings.Trim(cells[9].Value, " "),
			CheckFinding: strings.Trim(cells[15].Value, " "),
			//AdmissionNumber: strings.Trim(cells[16].Value, " "),
			CreateTime: time.Now().Format("2006-01-02 15:04:05"),
			UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
		}
		if len(cells) >= 17 {
			dataInfo.AdmissionNumber = strings.Trim(cells[16].Value, " ")
		}
		if len(dataInfo.Name) <= 0 && len(dataInfo.VisitCardID) <= 0 {
			fmt.Printf("姓名和就诊卡号均为空，直接过滤[deal_error] \n")
			continue
		}
		if len(dataInfo.Name) <= 0 || len(dataInfo.VisitCardID) <= 0 {
			fmt.Printf("姓名：%s 或就诊卡号 :%s,为空\n", dataInfo.Name, dataInfo.VisitCardID)
			//continue
		}
		dataList = append(dataList, dataInfo)
		if i > 2 && i%100 == 0 { // 每100条写入一次 并重置切片
			err := m.db.Table(tables.TableBcV2).CreateInBatches(dataList, len(dataList)).Error
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
		err := m.db.Table(tables.TableBcV2).CreateInBatches(dataList, len(dataList)).Error
		if err != nil {
			fmt.Printf("db get 异常 err:%s", err.Error())
			return err
		}
		total += len(dataList)
		//fmt.Printf("入库:%d 条,total:%d\n", len(collectList), total)
		dataList = dataList[0:0]
	}
	fmt.Printf("B超表入库成功，记录数：%d\n", total)
	return nil
}

func (m *BcV2Service) Merge() (err error) {
	// 遍历数据，获取符合条件的总表数据 若找到则选择一条合适的填充，否则打印提示并继续处理下一条
	total := 0
	pageSize := 1000
	pageIndex := 1
	for {
		dataList := make([]*tables.TBcV2, 0)
		err = m.db.Table(tables.TableBcV2).Select(tables.TableBcV2Fields).Order("name asc,visitCardId asc,visitTime asc").
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
			//fmt.Printf("QueryTCollect break!count:%d,pageSize:%d ", count, pageSize)
			break
		}
	}
	if total > 0 {
		fmt.Printf("B超表和入主表完成，匹配成功【%d】，匹配不到【%d】，匹配冲突【%d】，系统异常【%d】\n",
			m.mergeSuc, m.mergeFaild, m.mergeConflict, m.mergeErr)
	} else {
		fmt.Printf("本次无B超数据需要合入\n")
	}

	return nil
}

func (m *BcV2Service) mateAndWriteCollect(dataBcV2 *tables.TBcV2) (err error) {
	// 匹配多条改为只匹配最近一条 所以两者排序都换了方向 只找非冲突添加的记录进行合并或覆盖
	collectList, err := GetCollectList(m.db, dataBcV2.Name, dataBcV2.VisitCardID,
		dataBcV2.VisitTime-15, dataBcV2.VisitTime, "desc", "0")
	if err != nil {
		m.mergeErr++
		return err
	}
	if len(collectList) <= 0 {
		m.mergeFaild++
		fmt.Printf("B超数据找不到可合入的汇总数据！！姓名【%s】,就诊卡号[%s]\n", dataBcV2.Name,
			dataBcV2.VisitCardID)
		return nil
	}
	collectData := collectList[0]
	// 只用两列检查冲突即可 因为其他的都是这两列提取的
	if (len(collectData.F191) > 0 && collectData.F191 != dataBcV2.CheckResult) ||
		(len(collectData.F192) > 0 && collectData.F192 != dataBcV2.CheckFinding) {
		fmt.Printf("B超匹配冲突，新增！！总表id【%d】,表id【%d】姓名【%s】,就诊卡号[%s] \n",
			collectData.ID, dataBcV2.ID, dataBcV2.Name, dataBcV2.VisitCardID)
		newCollectData := &tables.TCollect{
			Name:        dataBcV2.Name,
			VisitCardID: dataBcV2.VisitCardID,
			VisitTime:   dataBcV2.VisitTime,
			F8:          dataBcV2.Age,
			F6:          dataBcV2.Sex,
			F191:        dataBcV2.CheckResult,
			F192:        dataBcV2.CheckFinding,
			F10:         dataBcV2.AdmissionNumber,
		}
		newCollectData.F193 = m.extractContent(F193Key1, dataBcV2.CheckFinding)
		if newCollectData.F193 == "" {
			newCollectData.F193 = m.extractContent(F193Key2, dataBcV2.CheckFinding)
			if newCollectData.F193 == "" {
				newCollectData.F193 = m.extractContent(F193Key3, dataBcV2.CheckFinding)
				if newCollectData.F193 == "" {
					newCollectData.F193 = m.extractContent(F193Key4, dataBcV2.CheckFinding)
				}
			}
		}
		newCollectData.F194 = m.extractContent(F194Key1, dataBcV2.CheckFinding)
		if newCollectData.F194 == "" {
			newCollectData.F194 = m.extractContent(F194Key2, dataBcV2.CheckFinding)
		}
		newCollectData.F195 = m.extractContent(F195Key1, dataBcV2.CheckFinding)
		if newCollectData.F195 == "" {
			newCollectData.F195 = m.extractContent(F195Key2, dataBcV2.CheckFinding)
		}
		newCollectData.F196 = m.extractContent(F196Key1, dataBcV2.CheckFinding)
		if newCollectData.F196 == "" {
			newCollectData.F196 = m.extractContent(F196Key2, dataBcV2.CheckFinding)
		}
		newCollectData.F197 = m.extractContent(F197Key1, dataBcV2.CheckFinding)
		if newCollectData.F197 == "" {
			newCollectData.F197 = m.extractContent(F197Key2, dataBcV2.CheckFinding)
		}
		newCollectData.F198 = m.extractContent(F198Key1, dataBcV2.CheckFinding)
		if newCollectData.F198 == "" {
			newCollectData.F198 = m.extractContent(F198Key2, dataBcV2.CheckFinding)
		}
		newCollectData.F199 = m.extractContent(F199Key1, dataBcV2.CheckFinding)
		if newCollectData.F199 == "" {
			newCollectData.F199 = m.extractContent(F199Key2, dataBcV2.CheckFinding)
		}
		newCollectData.F200 = m.extractContent(F200Key1, dataBcV2.CheckFinding)
		if newCollectData.F200 == "" {
			newCollectData.F200 = m.extractContent(F200Key2, dataBcV2.CheckFinding)
		}
		newCollectData.F247 = m.extractContent(F247Key1, dataBcV2.CheckFinding)
		if newCollectData.F247 == "" {
			newCollectData.F247 = m.extractContent(F247Key2, dataBcV2.CheckFinding)
			if newCollectData.F247 == "" {
				newCollectData.F247 = m.extractContent(F247Key3, dataBcV2.CheckFinding)
			}
		}
		newCollectData.F248 = m.extractContent(F248Key1, dataBcV2.CheckFinding)
		if newCollectData.F248 == "" {
			newCollectData.F248 = m.extractContent(F248Key2, dataBcV2.CheckFinding)
			if newCollectData.F248 == "" {
				newCollectData.F248 = m.extractContent(F248Key3, dataBcV2.CheckFinding)
			}
		}
		newCollectData.F249 = m.extractContent(F249Key1, dataBcV2.CheckFinding)
		if newCollectData.F249 == "" {
			newCollectData.F249 = m.extractContent(F249Key2, dataBcV2.CheckFinding)
			if newCollectData.F249 == "" {
				newCollectData.F249 = m.extractContent(F249Key3, dataBcV2.CheckFinding)
			}
		}

		err = AddCollect(m.db, newCollectData)
		if err != nil {
			m.mergeErr++
			fmt.Printf("B超数据 merge 异常！！姓名【%s】,就诊卡号[%s],新增汇总数据失败：%s\n", dataBcV2.Name,
				dataBcV2.VisitCardID, err.Error())
			return err
		}
		m.mergeConflict++
	} else {
		fmt.Printf("B超匹配成功，和入总表！！总表id【%d】,表id【%d】姓名【%s】,就诊卡号[%s] \n",
			collectData.ID, dataBcV2.ID, dataBcV2.Name, dataBcV2.VisitCardID)
		if len(collectData.F8) <= 0 {
			collectData.F8 = dataBcV2.Age
		}
		if len(collectData.F6) <= 0 {
			collectData.F6 = dataBcV2.Sex
		}
		collectData.F191 = dataBcV2.CheckResult
		collectData.F192 = dataBcV2.CheckFinding
		collectData.F10 = dataBcV2.AdmissionNumber

		collectData.F193 = m.extractContent(F193Key1, dataBcV2.CheckFinding)
		if collectData.F193 == "" {
			collectData.F193 = m.extractContent(F193Key2, dataBcV2.CheckFinding)
			if collectData.F193 == "" {
				collectData.F193 = m.extractContent(F193Key3, dataBcV2.CheckFinding)
				if collectData.F193 == "" {
					collectData.F193 = m.extractContent(F193Key4, dataBcV2.CheckFinding)
				}
			}
		}
		collectData.F194 = m.extractContent(F194Key1, dataBcV2.CheckFinding)
		if collectData.F194 == "" {
			collectData.F194 = m.extractContent(F194Key2, dataBcV2.CheckFinding)
		}
		collectData.F195 = m.extractContent(F195Key1, dataBcV2.CheckFinding)
		if collectData.F195 == "" {
			collectData.F195 = m.extractContent(F195Key2, dataBcV2.CheckFinding)
		}
		collectData.F196 = m.extractContent(F196Key1, dataBcV2.CheckFinding)
		if collectData.F196 == "" {
			collectData.F196 = m.extractContent(F196Key2, dataBcV2.CheckFinding)
		}
		collectData.F197 = m.extractContent(F197Key1, dataBcV2.CheckFinding)
		if collectData.F197 == "" {
			collectData.F197 = m.extractContent(F197Key2, dataBcV2.CheckFinding)
		}
		collectData.F198 = m.extractContent(F198Key1, dataBcV2.CheckFinding)
		if collectData.F198 == "" {
			collectData.F198 = m.extractContent(F198Key2, dataBcV2.CheckFinding)
		}
		collectData.F199 = m.extractContent(F199Key1, dataBcV2.CheckFinding)
		if collectData.F199 == "" {
			collectData.F199 = m.extractContent(F199Key2, dataBcV2.CheckFinding)
		}
		collectData.F200 = m.extractContent(F200Key1, dataBcV2.CheckFinding)
		if collectData.F200 == "" {
			collectData.F200 = m.extractContent(F200Key2, dataBcV2.CheckFinding)
		}
		collectData.F247 = m.extractContent(F247Key1, dataBcV2.CheckFinding)
		if collectData.F247 == "" {
			collectData.F247 = m.extractContent(F247Key2, dataBcV2.CheckFinding)
			if collectData.F247 == "" {
				collectData.F247 = m.extractContent(F247Key3, dataBcV2.CheckFinding)
			}
		}
		collectData.F248 = m.extractContent(F248Key1, dataBcV2.CheckFinding)
		if collectData.F248 == "" {
			collectData.F248 = m.extractContent(F248Key2, dataBcV2.CheckFinding)
			if collectData.F248 == "" {
				collectData.F248 = m.extractContent(F248Key3, dataBcV2.CheckFinding)
			}
		}
		collectData.F249 = m.extractContent(F249Key1, dataBcV2.CheckFinding)
		if collectData.F249 == "" {
			collectData.F249 = m.extractContent(F249Key2, dataBcV2.CheckFinding)
			if collectData.F249 == "" {
				collectData.F249 = m.extractContent(F249Key3, dataBcV2.CheckFinding)
			}
		}
		err = UpdateCollect(m.db, collectData)
		if err != nil {
			m.mergeErr++
			return err
		}
		m.mergeSuc++
	}

	return nil
}

func (m *BcV2Service) extractContent(specifiedKeyword, paragraph string) string {
	// 标准化段落中的空格字符
	paragraph = normalizeSpaces(paragraph)
	// 定义关键字集合
	keywords := map[string]bool{
		F193Key1: true,
		F193Key2: true,
		F193Key3: true,
		F193Key4: true,
		F194Key1: true,
		F194Key2: true,
		F195Key1: true,
		F195Key2: true,
		F196Key1: true,
		F196Key2: true,
		F197Key1: true,
		F197Key2: true,
		F198Key1: true,
		F198Key2: true,
		F199Key1: true,
		F199Key2: true,
		F200Key1: true,
		F200Key2: true,
		F247Key1: true,
		F247Key2: true,
		F247Key3: true,
		F248Key1: true,
		F248Key2: true,
		F248Key3: true,
		F249Key1: true,
		F249Key2: true,
		F249Key3: true,
	}
	// 检查指定关键字是否在关键字集合中
	if !keywords[specifiedKeyword] {
		fmt.Printf("specifiedKeyword【%s】 not in map\n", specifiedKeyword)
		return ""
	}

	// 搜索指定关键字在段落中的位置
	startIndex := strings.Index(paragraph, specifiedKeyword)
	if startIndex == -1 {
		//fmt.Printf("specifiedKeyword【%s】 not found,paragraph:%s\n", specifiedKeyword, paragraph)
		return ""
	}

	// 从指定关键字之后搜索下一个关键字的位置
	endIndex := len(paragraph) // 默认到段落结尾
	foundEndIndex := endIndex
	for keyword, _ := range keywords {
		if keyword == specifiedKeyword {
			continue
		}
		index := strings.Index(paragraph[startIndex+len(specifiedKeyword):], keyword)
		if index != -1 {
			foundEndIndex = startIndex + len(specifiedKeyword) + index
			//fmt.Printf("specifiedKeyword【%s】 nextKey[%s] foundEndIndex[%d] found\n", specifiedKeyword, keyword, foundEndIndex)
			if foundEndIndex < endIndex {
				//fmt.Printf("找到了更近的关键字【%s】", keyword)
				endIndex = foundEndIndex
			}
		}
	}

	// 返回从指定关键字开始到下一个关键字之间的内容
	return paragraph[startIndex:endIndex]
}

// 标准化空格字符
func normalizeSpaces(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, s)
}
