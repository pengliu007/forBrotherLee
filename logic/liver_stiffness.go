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

// 肝硬度
type LiverStiffnessService struct {
	//fileHandle *xlsx.File
	mergeErr      int
	mergeSuc      int
	mergeConflict int
	mergeFaild    int
	db            *gorm.DB
}

func NewLiverStiffnessService() *LiverStiffnessService {
	return &LiverStiffnessService{}
}

func (m *LiverStiffnessService) InitDb() (err error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/inspectionInfo?charset=utf8mb4&parseTime=false&maxAllowedPacket=104857600"
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	//m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("new db 异常 failed, err:%+v", err)
		return err
	}

	sqlStr := "truncate table " + tables.TableLiverStiffness
	err = m.db.Exec(sqlStr, []interface{}{}...).Error
	if err != nil {
		fmt.Printf("清空消化科肝硬度touch表失败异常")
		return err
	}
	return nil
}

func (m *LiverStiffnessService) LoadFile(fileName string) (err error) {
	fileHandle, err := xlsx.OpenFile(fileName)
	if err != nil {
		fmt.Printf("肝硬度touch表打开失败异常,err:%s", err.Error())
		return err
	}
	fmt.Printf("肝硬度touch表打开成功，记录数：%d\n", len(fileHandle.Sheets[0].Rows))

	// 入库
	dataList := make([]*tables.TLiverStiffness, 0)
	total := 0
	for i, rowInfo := range fileHandle.Sheets[0].Rows {
		if i < 2 { //前两行无需入库
			fmt.Printf("前两行为表头无需入库:%d\n", i)
			continue
		}
		cells := rowInfo.Cells
		if len(cells) < 28 {
			fmt.Printf("cells [异常] len[%d]，err\n", len(cells))
			continue
		}
		visitTime, err := strconv.ParseFloat(strings.Trim(cells[7].Value, " "), 64)
		if err != nil {
			fmt.Printf("肝硬度touch表表记录就诊时间异常：姓名:%s，卡号:%s,时间【%s】\n",
				cells[0].Value, cells[3].Value, cells[7].Value)
			return err
		}
		dataInfo := &tables.TLiverStiffness{
			Name:                             strings.Trim(cells[0].Value, " "),
			VisitCardID:                      strings.Trim(cells[3].Value, " "),
			VisitTime:                        int(visitTime),
			FiberScansSucNum:                 cells[9].Value,
			FatAttenuation:                   cells[10].Value,
			FatAttenuationQuartileDifference: cells[11].Value,
			Hardness:                         cells[12].Value,
			HardnessQuartileDifference:       cells[13].Value,
			FiberScansTotalNum:               cells[14].Value,
			Phone:                            cells[15].Value,
			Height:                           cells[16].Value,
			Weight:                           cells[17].Value,
			DetectionDuration:                cells[18].Value,
			SucNums:                          cells[27].Value,
			CreateTime:                       time.Now().Format("2006-01-02 15:04:05"),
			UpdateTime:                       time.Now().Format("2006-01-02 15:04:05"),
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
			err := m.db.Table(tables.TableLiverStiffness).CreateInBatches(dataList, len(dataList)).Error
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
		err := m.db.Table(tables.TableLiverStiffness).CreateInBatches(dataList, len(dataList)).Error
		if err != nil {
			fmt.Printf("db get 异常 err:%s", err.Error())
			return err
		}
		total += len(dataList)
		//fmt.Printf("入库:%d 条,total:%d\n", len(collectList), total)
		dataList = dataList[0:0]
	}
	fmt.Printf("肝硬度touch表入库成功，记录数：%d\n", total)
	return nil
}

func (m *LiverStiffnessService) Merge() (err error) {
	// 遍历数据，获取符合条件的总表数据 若找到则选择一条合适的填充，否则打印提示并继续处理下一条
	total := 0
	pageSize := 1000
	pageIndex := 1
	for {
		dataList := make([]*tables.TLiverStiffness, 0)
		err = m.db.Table(tables.TableLiverStiffness).Select(tables.TableLiverStiffnessFields).Order("name asc,visitCardId asc,visitTime desc").
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
		fmt.Printf("肝硬度表和入主表完成，匹配成功【%d】，匹配不到【%d】，匹配冲突【%d】，系统异常【%d】\n",
			m.mergeSuc, m.mergeFaild, m.mergeConflict, m.mergeErr)
	} else {
		fmt.Printf("本次无肝硬度数据需要合入\n")
	}

	return nil
}

func (m *LiverStiffnessService) mateAndWriteCollect(dataLiverStiffness *tables.TLiverStiffness) (err error) {
	collectList, err := GetCollectList(m.db, dataLiverStiffness.Name, dataLiverStiffness.VisitCardID,
		dataLiverStiffness.VisitTime-14, dataLiverStiffness.VisitTime+14, "asc")
	if err != nil {
		m.mergeErr++
		return err
	}
	if len(collectList) <= 0 {
		m.mergeFaild++
		fmt.Printf("肝硬度touch数据找不到可合入的汇总数据！！姓名【%s】,就诊卡号[%s]\n", dataLiverStiffness.Name,
			dataLiverStiffness.VisitCardID)
		return nil
	}
	for i, collectData := range collectList {
		if len(collectData.F180) > 0 || len(collectData.F181) > 0 || len(collectData.F182) > 0 ||
			len(collectData.F183) > 0 || len(collectData.F184) > 0 || len(collectData.F185) > 0 ||
			len(collectData.F189) > 0 || len(collectData.F190) > 0 {
			fmt.Printf("肝硬度touch匹配到到总表数据共【%d】条，第【%d】条已存在监测记录，继续匹配！！姓名【%s】,"+
				"就诊卡号[%s] 总表时间【%d】,touch表时间【%d】\n", len(collectList), i+1, dataLiverStiffness.Name,
				dataLiverStiffness.VisitCardID, collectData.VisitTime, dataLiverStiffness.VisitTime)
			m.mergeConflict++
			if i == len(collectList)-1 {
				fmt.Printf("肝硬度touch匹配到到总表数据共【%d】条 全都冲突，匹配失败！！姓名【%s】,"+
					"就诊卡号[%s] 总表时间【%d】,touch表时间【%d】\n", len(collectList), dataLiverStiffness.Name,
					dataLiverStiffness.VisitCardID, collectData.VisitTime, dataLiverStiffness.VisitTime)
				m.mergeFaild++
			}
			continue
		}
		collectData.F180 = dataLiverStiffness.FiberScansSucNum
		collectData.F181 = dataLiverStiffness.FatAttenuation
		collectData.F182 = dataLiverStiffness.FatAttenuationQuartileDifference
		collectData.F183 = dataLiverStiffness.Hardness
		collectData.F184 = dataLiverStiffness.HardnessQuartileDifference
		collectData.F185 = dataLiverStiffness.FiberScansTotalNum
		collectData.F186 = dataLiverStiffness.Phone
		collectData.F187 = dataLiverStiffness.Height
		collectData.F188 = dataLiverStiffness.Weight
		collectData.F189 = dataLiverStiffness.DetectionDuration
		collectData.F190 = dataLiverStiffness.SucNums

		err = UpdateCollect(m.db, collectData)
		if err != nil {
			m.mergeErr++
			return err
		}
		m.mergeSuc++
		fmt.Printf("肝硬度touch匹配成功，和入总表！！总表id【%d】,touch表id【%d】姓名【%s】,就诊卡号[%s] \n",
			collectData.ID, dataLiverStiffness.ID, dataLiverStiffness.Name, dataLiverStiffness.VisitCardID)
	}

	return nil
}
