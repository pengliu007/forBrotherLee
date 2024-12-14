package logic

import (
	"bufio"
	"errors"
	"fmt"
	"strconv"
	//"github.com/tealeg/xlsx"
	//"gorm.io/gorm"
	"os"
)

type ScriptService struct {
	isLoadedUser    bool
	isLoadedCollect bool
	resultFileNum   int
	//userInfoService       *UserInfoService
	collectService        *CollectService
	liverStiffnessService *LiverStiffnessService // 消化
	laboratoryService     *LaboratoryService     // 检验科
	urineCultureService   *UrineCultureService   // 尿培养
	pathologyService      *PathologyService      // 病理
	bcService             *BcService             // b超
}

func NewScriptService() *ScriptService {
	return &ScriptService{
		resultFileNum: 1,
		//	userInfoService:       NewUserInfoService(),
		collectService:        NewCollectService(),
		liverStiffnessService: NewLiverStiffnessService(),
		laboratoryService:     NewLaboratoryService(),
		urineCultureService:   NewUrineCultureService(),
		pathologyService:      NewPathologyService(),
		bcService:             NewBcService(),
	}
}

func (s *ScriptService) RunTask(args []string) error {
	// 初始化所有处理实体db链接
	err := s.InitDb()
	if err != nil {
		return err
	}
	// 解析输入并按输入加载excel表 将表格数据入库  时间格式时excel格式，可直接用来做比较等，输出时做下处理即可
	err = s.DealInput()
	if err != nil {
		fmt.Printf("读取解析文件异常，err：%s\n", err.Error())
		return err
	}
	// 填充检验数据都到总表
	err = s.laboratoryService.Merge()
	if err != nil {
		fmt.Printf("填充检验数据都到总表异常，err：%s\n", err.Error())
		return err
	}
	// 填充尿培养数据到总表
	err = s.urineCultureService.Merge()
	if err != nil {
		fmt.Printf("填充尿培养数据到总表异常，err：%s\n", err.Error())
		return err
	}
	// 填充病理数据到总表
	err = s.pathologyService.Merge()
	if err != nil {
		fmt.Printf("填充病理数据到总表异常，err：%s\n", err.Error())
		return err
	}
	// 填充b超数据到总表
	err = s.bcService.Merge()
	if err != nil {
		fmt.Printf("填充b超数据到总表异常，err：%s\n", err.Error())
		return err
	}
	// 填充消化数据到总表 500touch 要最后
	err = s.liverStiffnessService.Merge()
	if err != nil {
		fmt.Printf("填充消化数据到总表[500touch]异常，err：%s\n", err.Error())
		return err
	}
	// 输出总表数据到新excel文件 出库时先把前两行出了  剩下的在做排序等
	err = s.collectService.MakeNewCollectExcel(s.resultFileNum)
	if err != nil {
		fmt.Printf("生成新汇总文件异常，err：%s\n", err.Error())
		return err
	}
	return nil
}
func (s *ScriptService) InitDb() error {
	//err := s.userInfoService.InitDb()
	//if err != nil {
	//	return err
	//}
	err := s.collectService.InitDb()
	if err != nil {
		return err
	}
	// 检验
	err = s.laboratoryService.InitDb()
	if err != nil {
		return err
	}
	// 消化肝硬度touch500
	err = s.liverStiffnessService.InitDb()
	if err != nil {
		return err
	}
	err = s.urineCultureService.InitDb()
	if err != nil {
		return err
	}

	err = s.pathologyService.InitDb()
	if err != nil {
		return err
	}

	err = s.bcService.InitDb()
	if err != nil {
		return err
	}
	return nil

}

func (s *ScriptService) DealInput() (err error) {
	for {
		if len(os.Args) >= 3 && os.Args[1] == "test" {
			err = s.doTestLoad(os.Args)
			if err != nil {
				return err
			}
			break
		}
		inputPrompt := "请输入要导入的文件类型并按回车结束(请务必确保用户信息，总表录入，其他检查表可按需录入!)" +
			"【1：总表；2：检验科表；3：尿培养表；4：消化科肝硬度touch表；5：病理表；6：B超表；0：结束所有输入启动合成任务】\n"

		fileType := getStdinInput(inputPrompt)
		if fileType == "" {
			fmt.Printf("%s", inputPrompt)
			continue
		}
		if fileType == "0" {
			if !s.isLoadedCollect {
				fmt.Println("用户信息表或总表还未录入无法执行任务,请继续录入\n")
				continue
			}
			resultFileNum := getStdinInput("您已完成文件导入，请输入需要将结果分几个文件输出：")
			s.resultFileNum, _ = strconv.Atoi(resultFileNum)
			if s.resultFileNum <= 0 {
				s.resultFileNum = 1
			}
			fmt.Println("您已完成文件导入，请输入需要将结果分%s个文件输出,实际按%d个输出，请耐心等待任务执行：\n",
				resultFileNum, s.resultFileNum)
			break
		}
		fileName := getStdinInput("请输入要导入的文件名并按回车结束 \n")
		if fileType == "" {
			fmt.Println("请输入要导入的文件名并按回车结束\n")
			continue
		}
		err = s.loadFile(fileType, fileName)
		if err != nil {
			return err
		}

	}
	return nil
}

func getStdinInput(hint string) string {
	fmt.Print(hint)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text()
	}
	return ""
}

func (s *ScriptService) doTestLoad(args []string) (err error) {
	s.resultFileNum, err = strconv.Atoi(os.Args[2])
	for i := 3; i+1 < len(os.Args); i += 2 {
		err = s.loadFile(os.Args[i], os.Args[i+1])
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ScriptService) loadFile(fileType string, name string) (err error) {
	if fileType == "1" {
		err = s.collectService.LoadFile(name)
		if err != nil {
			fmt.Printf("汇总表加载失败,err:%s", err.Error())
			return err
		}
		s.isLoadedCollect = true
	} else if fileType == "2" {
		err = s.laboratoryService.LoadFile(name)
		if err != nil {
			fmt.Printf("检验表加载失败,err:%s", err.Error())
			return err
		}
	} else if fileType == "3" {
		err = s.urineCultureService.LoadFile(name)
		if err != nil {
			fmt.Printf("尿培养文件加载失败,err:%s", err.Error())
			return err
		}
	} else if fileType == "4" {
		err = s.liverStiffnessService.LoadFile(name)
		if err != nil {
			fmt.Printf("消化肝硬度touch500加载失败,err:%s", err.Error())
			return err
		}
	} else if fileType == "5" {
		err = s.pathologyService.LoadFile(name)
		if err != nil {
			fmt.Printf("病理文件加载失败,err:%s", err.Error())
			return err
		}
	} else if fileType == "6" {
		err = s.bcService.LoadFile(name)
		if err != nil {
			fmt.Printf("b超文件加载失败,err:%s", err.Error())
			return err
		}
	} else {
		fmt.Printf("不支持的文件类型，请重新输入")
		return errors.New("不支持的文件类型，请重新输入")
	}
	return nil
}
