package logic

import (
	"errors"
	"fmt"
	"github.com/pengliu007/forBrotherLee/tables"
	"gorm.io/gorm"
	"time"
)

func GetCollectList(db *gorm.DB, name string, visitCardID string, beginTime int, endTime int, order string,
	isConflict string) (
	collectList []*tables.TCollect, err error) {
	collectList = make([]*tables.TCollect, 0)
	orderStr := "visitTime " + order
	db = db.Table(tables.TableCollect).Select(tables.TableCollectFields)
	db = db.Where("name", name).Where("visitCardID", visitCardID).Where("visitTime>=?", beginTime).
		Where("visitTime<=?", endTime)
	if len(isConflict) > 0 {
		db = db.Where("isConflict", isConflict)
	}
	err = db.Order(orderStr).Find(&collectList).Error
	if nil != err {
		fmt.Printf("获取总表数据异常 GetCollectList err：%s\n", err.Error())
		return nil, err
	}

	return collectList, nil
}

func AddCollect(db *gorm.DB, collectInfo *tables.TCollect) (err error) {
	collectInfo.CreateTime = time.Now().Format("2006-01-02 15:04:05")
	collectInfo.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	err = db.Table(tables.TableCollect).Create(collectInfo).Scan(collectInfo).Error
	if err != nil {
		fmt.Printf("写入总表数据异常 AddCollect err：%s\n", err.Error())
		return err
	}
	return nil
}

func UpdateCollect(db *gorm.DB, collectInfo *tables.TCollect) (err error) {
	if collectInfo.ID <= 0 {
		fmt.Printf("UpdateCollect id empty err 异常!!! ")
		return errors.New("UpdateCollect id empty err!!!")
	}
	collectInfo.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	db = db.Table(tables.TableCollect)
	where := map[string]interface{}{
		"id": collectInfo.ID,
	}
	db = db.Where(where)
	err = db.Limit(1).Updates(collectInfo).Error
	return nil
}

func UpdateCollectByCond(db *gorm.DB, param, where map[string]interface{}) (err error) {
	if len(param) <= 0 || len(where) <= 0 {
		errors.New("updateCollectByCond param or where empty err")
	}
	err = db.Table(tables.TableCollect).Where(where).Updates(param).Error

	return err
}
