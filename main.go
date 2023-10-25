package main

import (
	"fmt"
	"github.com/pengliu007/forBrotherLee/logic"
	"github.com/tealeg/xlsx"
	"os"
)

func main() {
	service := logic.NewScriptService()
	err := service.RunTask(os.Args)
	if err != nil {
		return
		//os.Exit(1)
	}
	//justTest()
}

func justTest() {
	newFile := xlsx.NewFile()
	sheet, _ := newFile.AddSheet("就是测试一下")
	headRow := sheet.AddRow()
	cell := headRow.AddCell()
	cell.Value = "ID"
	cell = headRow.AddCell()
	cell.Value = "名称"

	firstRow := sheet.AddRow()
	cell = firstRow.AddCell()
	cell.Value = "1"
	cell = firstRow.AddCell()
	cell.Value = "帅气的刘小鹏呀"
	err := newFile.Save("newfile.xlsx")
	if err != nil {
		fmt.Printf("结果文件保存失败,err:%s", err.Error())
		os.Exit(1)
	}
}
