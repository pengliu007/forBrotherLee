package logic

import (
	"fmt"
	"github.com/pengliu007/forBrotherLee/tables"
	"github.com/tealeg/xlsx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"

	"strconv"
	"strings"
)

// B超
type BcService struct {
	//fileHandle *xlsx.File
	mergeErr      int
	mergeSuc      int
	mergeConflict int
	mergeFaild    int
	db            *gorm.DB
}

func NewBcService() *BcService {
	return &BcService{}
}

func (m *BcService) InitDb() (err error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/inspectionInfo?charset=utf8mb4&parseTime=false&maxAllowedPacket=104857600"
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	//m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("new db 异常 failed, err:%+v", err)
		return err
	}

	sqlStr := "truncate table " + tables.TableBc
	err = m.db.Exec(sqlStr, []interface{}{}...).Error
	if err != nil {
		fmt.Printf("清空消化科B超表失败异常")
		return err
	}
	return nil
}

func (m *BcService) LoadFile(fileName string) (err error) {
	fileHandle, err := xlsx.OpenFile(fileName)
	if err != nil {
		fmt.Printf("B超表打开失败异常,err:%s", err.Error())
		return err
	}
	fmt.Printf("B超表打开成功，记录数：%d\n", len(fileHandle.Sheets[0].Rows))

	// 入库
	dataList := make([]*tables.TBc, 0)
	total := 0
	for i, rowInfo := range fileHandle.Sheets[0].Rows {
		if i < 1 { //前1行无需入库
			fmt.Printf("前两行为表头无需入库:%d\n", i)
			continue
		}
		cells := rowInfo.Cells
		if len(cells) < 8 {
			fmt.Printf("cells [异常] len[%d]，err\n", len(cells))
			continue
		}

		timeSrc := strings.Trim(cells[1].Value, " ")
		var visitTime float64
		if len(timeSrc) == 10 {
			timeGo, err := time.Parse("2006-01-02", timeSrc)
			if err != nil {
				fmt.Printf("B超表表记录就诊时间异常1：姓名:%s，卡号:%s,时间【%s】\n",
					cells[2].Value, cells[6].Value, cells[1].Value)
				return err
			}
			visitTime = xlsx.TimeToExcelTime(timeGo, false)
		} else {
			if len(timeSrc) <= 0 {
				fmt.Printf("B超表表记录就诊时间异常,忽律此条：姓名:%s，卡号:%s,时间【%s】\n",
					cells[2].Value, cells[6].Value, cells[1].Value)
				continue
			}
			visitTime, err = strconv.ParseFloat(timeSrc, 64)
			if err != nil {
				fmt.Printf("B超表表记录就诊时间异常2：姓名:%s，卡号:%s,时间【%s】\n",
					cells[2].Value, cells[6].Value, cells[1].Value)
				return err
			}
		}

		dataInfo := &tables.TBc{
			Name:            strings.Trim(cells[2].Value, " "),
			VisitCardID:     strings.Trim(cells[6].Value, " "),
			VisitTime:       int(visitTime),
			Age:             strings.Replace(strings.Trim(cells[4].Value, " "), "岁", "", -1),
			Sex:             strings.Trim(cells[3].Value, " "),
			CheckResult:     strings.Trim(cells[5].Value, " "),
			AdmissionNumber: strings.Trim(cells[7].Value, " "),
			CreateTime:      time.Now().Format("2006-01-02 15:04:05"),
			UpdateTime:      time.Now().Format("2006-01-02 15:04:05"),
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
			err := m.db.Table(tables.TableBc).CreateInBatches(dataList, len(dataList)).Error
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
		err := m.db.Table(tables.TableBc).CreateInBatches(dataList, len(dataList)).Error
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

func (m *BcService) Merge() (err error) {
	// 遍历数据，获取符合条件的总表数据 若找到则选择一条合适的填充，否则打印提示并继续处理下一条
	total := 0
	pageSize := 1000
	pageIndex := 1
	for {
		dataList := make([]*tables.TBc, 0)
		err = m.db.Table(tables.TableBc).Select(tables.TableBcFields).Order("name asc,visitCardId asc,visitTime asc").
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
		fmt.Printf("B超表和入主表完成，匹配成功【%d】，匹配不到【%d】，匹配冲突【%d】，系统异常【%d】\n",
			m.mergeSuc, m.mergeFaild, m.mergeConflict, m.mergeErr)
	} else {
		fmt.Printf("本次无B超数据需要合入\n")
	}

	return nil
}

func (m *BcService) mateAndWriteCollect(dataBc *tables.TBc) (err error) {
	// 匹配多条改为只匹配最近一条 所以两者排序都换了方向 只找非冲突添加的记录进行合并或覆盖
	collectList, err := GetCollectList(m.db, dataBc.Name, dataBc.VisitCardID,
		dataBc.VisitTime-15, dataBc.VisitTime, "desc", "0")
	if err != nil {
		m.mergeErr++
		return err
	}
	if len(collectList) <= 0 {
		m.mergeFaild++
		fmt.Printf("B超数据找不到可合入的汇总数据！！姓名【%s】,就诊卡号[%s]\n", dataBc.Name,
			dataBc.VisitCardID)
		return nil
	}
	collectData := collectList[0]
	if len(collectData.F191) > 0 && collectData.F191 != dataBc.CheckResult {
		fmt.Printf("B超匹配冲突，新增！！总表id【%d】,表id【%d】姓名【%s】,就诊卡号[%s] \n",
			collectData.ID, dataBc.ID, dataBc.Name, dataBc.VisitCardID)
		newCollectData := &tables.TCollect{
			Name:        dataBc.Name,
			VisitCardID: dataBc.VisitCardID,
			VisitTime:   dataBc.VisitTime,
			F8:          dataBc.Age,
			F6:          dataBc.Sex,
			F191:        dataBc.CheckResult,
			F10:         dataBc.AdmissionNumber,
		}
		err = AddCollect(m.db, newCollectData)
		if err != nil {
			m.mergeErr++
			fmt.Printf("B超数据 merge 异常！！姓名【%s】,就诊卡号[%s],新增汇总数据失败：%s\n", dataBc.Name,
				dataBc.VisitCardID, err.Error())
			return err
		}
		m.mergeConflict++
	} else {
		fmt.Printf("B超匹配成功，和入总表！！总表id【%d】,表id【%d】姓名【%s】,就诊卡号[%s] \n",
			collectData.ID, dataBc.ID, dataBc.Name, dataBc.VisitCardID)
		collectData.F191 = dataBc.CheckResult
		collectData.F10 = dataBc.AdmissionNumber
		if len(collectData.F8) <= 0 {
			collectData.F8 = dataBc.Age
		}
		if len(collectData.F6) <= 0 {
			collectData.F6 = dataBc.Sex
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
