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

// 病理
type PathologyService struct {
	//fileHandle *xlsx.File
	mergeErr      int
	mergeSuc      int
	mergeConflict int
	mergeFaild    int
	db            *gorm.DB
}

func NewPathologyService() *PathologyService {
	return &PathologyService{}
}

func (m *PathologyService) InitDb() (err error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/inspectionInfo?charset=utf8mb4&parseTime=false&maxAllowedPacket=104857600"
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	//m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("new db 异常 failed, err:%+v", err)
		return err
	}

	sqlStr := "truncate table " + tables.TablePathology
	err = m.db.Exec(sqlStr, []interface{}{}...).Error
	if err != nil {
		fmt.Printf("清空病理表失败异常")
		return err
	}
	return nil
}

func (m *PathologyService) LoadFile(fileName string) (err error) {
	fileHandle, err := xlsx.OpenFile(fileName)
	if err != nil {
		fmt.Printf("病理表打开失败异常,err:%s", err.Error())
		return err
	}
	fmt.Printf("病理表打开成功，记录数：%d\n", len(fileHandle.Sheets[0].Rows))

	// 入库
	dataList := make([]*tables.TPathology, 0)
	total := 0
	for i, rowInfo := range fileHandle.Sheets[0].Rows {
		if i < 1 { //前1行无需入库
			fmt.Printf("前1行为表头无需入库:%d\n", i)
			continue
		}
		cells := rowInfo.Cells
		if len(cells) < 11 {
			fmt.Printf("cells [异常] len[%d]，err\n", len(cells))
			continue
		}
		visitTime, err := strconv.ParseFloat(strings.Trim(cells[10].Value, " "), 64)
		if err != nil {
			fmt.Printf("病理表表记录就诊时间异常：姓名:%s，卡号:%s,时间【%s】\n",
				cells[3].Value, cells[1].Value, cells[10].Value)
			//return err
			continue
		}
		pathologyTime, err := strconv.ParseFloat(strings.Trim(cells[9].Value, " "), 64)
		if err != nil {
			fmt.Printf("病理表表记录肝穿时间异常：姓名:%s，卡号:%s,时间【%s】\n",
				cells[3].Value, cells[1].Value, cells[9].Value)
			pathologyTime = 0
		}

		dataInfo := &tables.TPathology{
			Name:            strings.Trim(cells[3].Value, " "),
			VisitCardID:     strings.Trim(cells[1].Value, " "),
			VisitTime:       int(visitTime),
			PathologyID:     strings.Trim(cells[0].Value, " "),
			AdmissionNumber: strings.Trim(cells[2].Value, " "),
			Sex:             strings.Trim(cells[4].Value, " "),
			Age:             strings.Replace(strings.Trim(cells[5].Value, " "), "岁", "", -1),
			PathologyResult: cells[8].Value,
			PathologyTime:   int(pathologyTime),

			CreateTime: time.Now().Format("2006-01-02 15:04:05"),
			UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
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
		if i > 1 && i%100 == 0 { // 每100条写入一次 并重置切片
			err := m.db.Table(tables.TablePathology).CreateInBatches(dataList, len(dataList)).Error
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
		err := m.db.Table(tables.TablePathology).CreateInBatches(dataList, len(dataList)).Error
		if err != nil {
			fmt.Printf("db get 异常 err:%s", err.Error())
			return err
		}
		total += len(dataList)
		//fmt.Printf("入库:%d 条,total:%d\n", len(collectList), total)
		dataList = dataList[0:0]
	}
	fmt.Printf("病理表入库成功，记录数：%d\n", total)
	return nil
}

func (m *PathologyService) Merge() (err error) {
	// 遍历数据，获取符合条件的总表数据 若找到则选择一条合适的填充，否则打印提示并继续处理下一条
	total := 0
	pageSize := 1000
	pageIndex := 1
	for {
		dataList := make([]*tables.TPathology, 0)
		err = m.db.Table(tables.TablePathology).Select(tables.TablePathologyFields).Order("name asc,visitCardId asc,visitTime asc").
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
		fmt.Printf("病理表和入主表完成，匹配成功【%d】，匹配不到【%d】，匹配冲突【%d】，系统异常【%d】\n",
			m.mergeSuc, m.mergeFaild, m.mergeConflict, m.mergeErr)
	} else {
		fmt.Printf("本次无病理数据需要合入\n")
	}

	return nil
}

func (m *PathologyService) mateAndWriteCollect(dataPathology *tables.TPathology) (err error) {
	// 匹配多条改为只匹配最近一条 所以两者排序都换了方向 只找非冲突添加的记录进行合并或覆盖
	collectList, err := GetCollectList(m.db, dataPathology.Name, dataPathology.VisitCardID,
		dataPathology.VisitTime-15, dataPathology.VisitTime, "desc", "0")
	if err != nil {
		m.mergeErr++
		return err
	}
	if len(collectList) <= 0 {
		m.mergeFaild++
		fmt.Printf("病理数据找不到可合入的汇总数据！！姓名【%s】,就诊卡号[%s]\n", dataPathology.Name,
			dataPathology.VisitCardID)
		return nil
	}
	// 解析三个特殊字段出来 F207(G) F208(S) F210(AIH评分)
	valueG, valueS, pathologyStages, valueAIH := m.getSpecialInfo(dataPathology.PathologyResult)
	for _, collectData := range collectList {
		if (len(collectData.F10) > 0 && collectData.F10 != dataPathology.AdmissionNumber) ||
			(collectData.F204 > 0 && collectData.F204 != dataPathology.PathologyTime) ||
			(len(collectData.F205) > 0 && collectData.F205 != dataPathology.PathologyID) ||
			(len(collectData.F206) > 0 && collectData.F206 != dataPathology.PathologyResult) ||
			(len(collectData.F207) > 0 && collectData.F207 != valueG) ||
			(len(collectData.F208) > 0 && collectData.F208 != valueS) ||
			(len(collectData.F209) > 0 && collectData.F209 != pathologyStages) ||
			(len(collectData.F210) > 0 && collectData.F210 != valueAIH) {
			fmt.Printf("病理匹配冲突，继续匹配！！总表id【%d】,病理表id【%d】姓名【%s】,就诊卡号[%s] \n",
				collectData.ID, dataPathology.ID, dataPathology.Name, dataPathology.VisitCardID)
			continue
		} else {
			fmt.Printf("病理匹配成功，和入总表！！总表id【%d】,病理表id【%d】姓名【%s】,就诊卡号[%s] \n",
				collectData.ID, dataPathology.ID, dataPathology.Name, dataPathology.VisitCardID)
			m.mergeSuc++
			if len(collectData.F8) <= 0 {
				collectData.F8 = dataPathology.Age
			}
			if len(collectData.F6) <= 0 {
				collectData.F6 = dataPathology.Sex
			}
			collectData.F205 = dataPathology.PathologyID
			collectData.F10 = dataPathology.AdmissionNumber
			collectData.F206 = dataPathology.PathologyResult
			collectData.F204 = dataPathology.PathologyTime
			collectData.F207 = valueG
			collectData.F208 = valueS
			collectData.F209 = pathologyStages
			collectData.F210 = valueAIH
			err = UpdateCollect(m.db, collectData)
			if err != nil {
				m.mergeErr++
				return err
			}
		}
		return nil
	}

	// 这里是都冲突了 丢弃不做处理
	m.mergeConflict++
	fmt.Printf("病理匹配冲突，丢弃！！病理表id【%d】姓名【%s】,就诊卡号[%s] \n",
		dataPathology.ID, dataPathology.Name, dataPathology.VisitCardID)

	return nil
}

func (m *PathologyService) getSpecialInfo(pathologyResult string) (valueG, valueS, pathologyStages, valueAIH string) {
	searchStr := pathologyResult
	// 先看G是否存在，是的话顺势解析g和s
	beginG := strings.Index(searchStr, "（G")
	if beginG >= 0 {
		searchStr = string([]byte(searchStr)[beginG:])
		endG := strings.Index(searchStr, "S")
		if endG >= 0 {
			// 这样取到的数据包含了搜索关键字，做一下replace
			valueG = string([]byte(searchStr)[0:endG])
			valueG = strings.Replace(valueG, "（G", "", -1)

			searchStr = string([]byte(searchStr)[endG+1:])
			endS := strings.Index(searchStr, "）")
			if endS >= 0 {
				valueS = string([]byte(searchStr)[0:endS])
			}
		}
	}
	// 重置searchStr 先找end 因为这个的begin太容易重复
	searchStr = pathologyResult
	endPathologyStages := strings.Index(searchStr, "期）")
	if endPathologyStages >= 0 {
		searchStr = string([]byte(searchStr)[0:endPathologyStages])
		// 往前找第一个（
		beginPathologyStages := strings.LastIndex(searchStr, "（")
		if beginPathologyStages >= 0 {
			searchStr = string([]byte(searchStr)[beginPathologyStages:])
			pathologyStages = strings.Replace(searchStr, "（", "", -1)
		}
	}
	// 重置searchStr
	searchStr = pathologyResult
	beginAIH := strings.Index(searchStr, "AIH评分")
	if beginAIH >= 0 {
		searchStr = string([]byte(searchStr)[beginAIH:])
		searchStr = strings.Replace(searchStr, "AIH评分", "", -1)
		endAIH := strings.Index(searchStr, "分")
		if endAIH >= 0 {
			valueAIH = string([]byte(searchStr)[0:endAIH])
		}
	}
	if len(valueAIH) <= 0 {
		// 重置searchStr
		searchStr = pathologyResult
		beginAIH := strings.Index(searchStr, "自身免疫性肝炎评分为")
		if beginAIH >= 0 {
			searchStr = string([]byte(searchStr)[beginAIH:])
			searchStr = strings.Replace(searchStr, "自身免疫性肝炎评分为", "", -1)
			endAIH := strings.Index(searchStr, "分")
			if endAIH >= 0 {
				valueAIH = string([]byte(searchStr)[0:endAIH])
			}
		}
	}
	return valueG, valueS, pathologyStages, valueAIH
}
