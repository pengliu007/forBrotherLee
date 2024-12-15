package tables

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	TableCollectFields = "id,name,visitCardId,f_1,f_2,VisitTime,f_5,f_6,f_7,f_8,f_10,f_11,f_12,f_13,f_14,f_15,f_16," +
		"f_17,f_18,f_19,f_20,f_21,f_22,f_23,f_24,f_25,f_26,f_27,f_28,f_29,f_30,f_31,f_32,f_33,f_34,f_35,f_36,f_37" +
		",f_38,f_39,f_40,f_41,f_42,f_43,f_44,f_45,f_46,f_47,f_48,f_49,f_50,f_51,f_52,f_53,f_54,f_55,f_56,f_57,f_58" +
		",f_59,f_60,f_61,f_62,f_63,f_64,f_65,f_66,f_67,f_68,f_69,f_70,f_71,f_72,f_73,f_74,f_75,f_76,f_77,f_78,f_79" +
		",f_80,f_81,f_82,f_83,f_84,f_85,f_86,f_87,f_88,f_89,f_90,f_91,f_92,f_93,f_94,f_95,f_96,f_97,f_98,f_99,f_100" +
		",f_101,f_102,f_103,f_104,f_105,f_106,f_107,f_108,f_109,f_110,f_111,f_112,f_113,f_114,f_115,f_116,f_117," +
		"f_118,f_119,f_120,f_121,f_122,f_123,f_124,f_125,f_126,f_127,f_128,f_129,f_130,f_131,f_132,f_133,f_134," +
		"f_135,f_136,f_137,f_138,f_139,f_140,f_141,f_142,f_143,f_144,f_145,f_146,f_147,f_148,f_149,f_150,f_151," +
		"f_152,f_153,f_154,f_155,f_156,f_157,f_158,f_159,f_160,f_161,f_162,f_163,f_164,f_165,f_166,f_167,f_168," +
		"f_169,f_170,f_171,f_172,f_173,f_174,f_175,f_176,f_177,f_178,f_179,f_180,f_181,f_182,f_183,f_184,f_185,f_186," +
		"f_187,f_188,f_189,f_190,f_191,f_192,f_193,f_194,f_195,f_196,f_197,f_198,f_199,f_200,f_201,f_202,f_203," +
		"f_204,f_205,f_206,f_207,f_208,f_209,f_210,f_211,f_212,f_213,f_214,f_215,f_216,f_217,f_218,f_219,f_220," +
		"f_221,f_222,f_223,f_224,f_225,f_226,f_227,f_228,f_229,f_230,f_231,f_232,f_233,f_234,f_235,f_236," +
		"f_237,f_238,f_239,f_240,f_241,f_242,f_243,f_244,f_245,f_246,f_247,f_248,f_249,IsConflict,createTime,updateTime "
)

// 总表
type TCollect struct {
	ID          int64  `json:"id" gorm:"column:id"`                   // 自增ID 策略id
	F1          string `json:"f_1" gorm:"column:f_1"`                 // 序号
	F2          string `json:"f_2" gorm:"column:f_2"`                 // 临床试验
	Name        string `json:"name" gorm:"column:name"`               // 患者姓名
	VisitTime   int    `json:"visitTime" gorm:"column:visitTime"`     // 就诊时间
	F5          string `json:"f_5" gorm:"column:f_5"`                 // 是否初治
	F6          string `json:"f_6" gorm:"column:f_6"`                 // 性别
	F7          string `json:"f_7" gorm:"column:f_7"`                 // 出生日期
	F8          string `json:"f_8" gorm:"column:f_8"`                 // 年龄
	VisitCardID string `json:"visitCardId" gorm:"column:visitCardId"` // 就诊卡ID
	F10         string `json:"f_10" gorm:"column:f_10"`               // 住院号
	F11         string `json:"f_11" gorm:"column:f_11"`               // 家庭住址
	F12         string `json:"f_12" gorm:"column:f_12"`               // 诊断
	F13         string `json:"f_13" gorm:"column:f_13"`               // 导入诊断
	F14         string `json:"f_14" gorm:"column:f_14"`               // 是否肝硬化
	F15         string `json:"f_15" gorm:"column:f_15"`               // 合并其他肝病
	F16         string `json:"f_16" gorm:"column:f_16"`               // 合并其他免疫疾病
	F17         string `json:"f_17" gorm:"column:f_17"`               // 合并代谢相关疾病
	F18         string `json:"f_18" gorm:"column:f_18"`               // 合并其他肿瘤
	F19         string `json:"f_19" gorm:"column:f_19"`               // 主诉
	F20         string `json:"f_20" gorm:"column:f_20"`               // 现病史
	F21         string `json:"f_21" gorm:"column:f_21"`               // 既往史
	F22         string `json:"f_22" gorm:"column:f_22"`               // 个人史
	F23         string `json:"f_23" gorm:"column:f_23"`               // 家族史
	F24         string `json:"f_24" gorm:"column:f_24"`               // 并发症
	F25         string `json:"f_25" gorm:"column:f_25"`               // 相关手术治疗
	F26         string `json:"f_26" gorm:"column:f_26"`               // 预后
	F27         string `json:"f_27" gorm:"column:f_27"`               // 白细胞计数
	F28         string `json:"f_28" gorm:"column:f_28"`               // 中性粒细胞绝对值
	F29         string `json:"f_29" gorm:"column:f_29"`               // 淋巴细胞绝对值
	F30         string `json:"f_30" gorm:"column:f_30"`               // 单核细胞绝对值
	F31         string `json:"f_31" gorm:"column:f_31"`               // 红细胞计数
	F32         string `json:"f_32" gorm:"column:f_32"`               // 血红蛋白
	F33         string `json:"f_33" gorm:"column:f_33"`               // 血细胞比容
	F34         string `json:"f_34" gorm:"column:f_34"`               // 平均红细胞体积
	F35         string `json:"f_35" gorm:"column:f_35"`               // 血小板计数
	F36         string `json:"f_36" gorm:"column:f_36"`               // 腺苷脱氨酶
	F37         string `json:"f_37" gorm:"column:f_37"`               // 线粒体型天门冬氨酸转氨酶
	F38         string `json:"f_38" gorm:"column:f_38"`               // 丙氨酸氨基转移酶(ALT)
	F39         string `json:"f_39" gorm:"column:f_39"`               // 天门冬氨酸转氨酶(AST)
	F40         string `json:"f_40" gorm:"column:f_40"`               // 总蛋白
	F41         string `json:"f_41" gorm:"column:f_41"`               // 球蛋白
	F42         string `json:"f_42" gorm:"column:f_42"`               // 白蛋白
	F43         string `json:"f_43" gorm:"column:f_43"`               // 总胆红素
	F44         string `json:"f_44" gorm:"column:f_44"`               // 直接胆红素
	F45         string `json:"f_45" gorm:"column:f_45"`               // 间接胆红素
	F46         string `json:"f_46" gorm:"column:f_46"`               // 碱性磷酸酶
	F47         string `json:"f_47" gorm:"column:f_47"`               // γ-谷氨酰基转移酶
	F48         string `json:"f_48" gorm:"column:f_48"`               // 总胆汁酸
	F49         string `json:"f_49" gorm:"column:f_49"`               // 亮氨酸氨基肽酶
	F50         string `json:"f_50" gorm:"column:f_50"`               // 白蛋白/球蛋白
	F51         string `json:"f_51" gorm:"column:f_51"`               // AST/ALT
	F52         string `json:"f_52" gorm:"column:f_52"`               // 总胆固醇
	F53         string `json:"f_53" gorm:"column:f_53"`               // 甘油三脂
	F54         string `json:"f_54" gorm:"column:f_54"`               // 高密度脂蛋白胆固醇
	F55         string `json:"f_55" gorm:"column:f_55"`               // 低密度脂蛋白胆固醇(LDL_C)
	F56         string `json:"f_56" gorm:"column:f_56"`               // 尿素
	F57         string `json:"f_57" gorm:"column:f_57"`               // 肌酐
	F58         string `json:"f_58" gorm:"column:f_58"`               // 葡萄糖
	F59         string `json:"f_59" gorm:"column:f_59"`               // 糖化血红蛋白
	F60         string `json:"f_60" gorm:"column:f_60"`               // 尿酸
	F61         string `json:"f_61" gorm:"column:f_61"`               // 载脂蛋白A1
	F62         string `json:"f_62" gorm:"column:f_62"`               // 载脂蛋白B
	F63         string `json:"f_63" gorm:"column:f_63"`               // 钾
	F64         string `json:"f_64" gorm:"column:f_64"`               // 钠
	F65         string `json:"f_65" gorm:"column:f_65"`               // 氯
	F66         string `json:"f_66" gorm:"column:f_66"`               // 总钙
	F67         string `json:"f_67" gorm:"column:f_67"`               // 二氧化碳
	F68         string `json:"f_68" gorm:"column:f_68"`               // 胱抑素C
	F69         string `json:"f_69" gorm:"column:f_69"`               // 无机磷
	F70         string `json:"f_70" gorm:"column:f_70"`               // 肌酸激酶
	F71         string `json:"f_71" gorm:"column:f_71"`               // 血清胆碱脂酶
	F72         string `json:"f_72" gorm:"column:f_72"`               // 凝血酶原时间
	F73         string `json:"f_73" gorm:"column:f_73"`               // 活化部分凝血活酶时间
	F74         string `json:"f_74" gorm:"column:f_74"`               // 纤维蛋白原含量
	F75         string `json:"f_75" gorm:"column:f_75"`               // 凝血酶时间
	F76         string `json:"f_76" gorm:"column:f_76"`               // D-二聚体
	F77         string `json:"f_77" gorm:"column:f_77"`               // 纤维蛋白原降解产物(血浆)
	F78         string `json:"f_78" gorm:"column:f_78"`               // 凝血酶原活动度
	F79         string `json:"f_79" gorm:"column:f_79"`               // PT国际标准化比值
	F80         string `json:"f_80" gorm:"column:f_80"`               // 血清免疫球蛋白G
	F81         string `json:"f_81" gorm:"column:f_81"`               // 血清免疫球蛋白A
	F82         string `json:"f_82" gorm:"column:f_82"`               // 血清免疫球蛋白M
	F83         string `json:"f_83" gorm:"column:f_83"`               // 血清免疫球蛋白E
	F84         string `json:"f_84" gorm:"column:f_84"`               // 免疫球蛋白G亚类4
	F85         string `json:"f_85" gorm:"column:f_85"`               // 免疫球蛋白G亚类3
	F86         string `json:"f_86" gorm:"column:f_86"`               // 免疫球蛋白G亚类2
	F87         string `json:"f_87" gorm:"column:f_87"`               // 免疫球蛋白G亚类1
	F88         string `json:"f_88" gorm:"column:f_88"`               // 超敏C反应蛋白
	F89         string `json:"f_89" gorm:"column:f_89"`               // 巨细胞病毒抗体IgM
	F90         string `json:"f_90" gorm:"column:f_90"`               // 抗EBV衣壳抗体IgG
	F91         string `json:"f_91" gorm:"column:f_91"`               // 抗EBV核抗体IgG
	F92         string `json:"f_92" gorm:"column:f_92"`               // 抗EBV早期抗体IgG
	F93         string `json:"f_93" gorm:"column:f_93"`               // 抗EBV衣壳抗体IgM
	F94         string `json:"f_94" gorm:"column:f_94"`               // 抗EBV壳抗体IgG亲合力
	F95         string `json:"f_95" gorm:"column:f_95"`               // EBV Zta蛋白抗体IgA
	F96         string `json:"f_96" gorm:"column:f_96"`               // 抗EBV衣壳抗体IgA
	F97         string `json:"f_97" gorm:"column:f_97"`               // 抗EBV核抗体IgA
	F98         string `json:"f_98" gorm:"column:f_98"`               // 异常凝血酶原
	F99         string `json:"f_99" gorm:"column:f_99"`               // 甲胎蛋白
	F100        string `json:"f_100" gorm:"column:f_100"`             // 糖链抗原CA19-9
	F101        string `json:"f_101" gorm:"column:f_101"`             // 糖链抗原CA125
	F102        string `json:"f_102" gorm:"column:f_102"`             // 癌胚抗原(CEA)
	F103        string `json:"f_103" gorm:"column:f_103"`             // 糖类抗原CA15-3
	F104        string `json:"f_104" gorm:"column:f_104"`             // 糖链抗原CA72-4
	F105        string `json:"f_105" gorm:"column:f_105"`             // 维生素D3
	F106        string `json:"f_106" gorm:"column:f_106"`             // 内毒素定量(血液)
	F107        string `json:"f_107" gorm:"column:f_107"`             // 1,3-β-D葡聚糖(血液)
	F108        string `json:"f_108" gorm:"column:f_108"`             // 血浆氨
	F109        string `json:"f_109" gorm:"column:f_109"`             // 快速血沉试验
	F110        string `json:"f_110" gorm:"column:f_110"`             // 游离三碘甲状腺原氨酸
	F111        string `json:"f_111" gorm:"column:f_111"`             // 游离甲状腺素
	F112        string `json:"f_112" gorm:"column:f_112"`             // 总三碘甲状腺原氨酸
	F113        string `json:"f_113" gorm:"column:f_113"`             // 总甲状腺素
	F114        string `json:"f_114" gorm:"column:f_114"`             // 促甲状腺激素
	F115        string `json:"f_115" gorm:"column:f_115"`             // 尿白细胞定性
	F116        string `json:"f_116" gorm:"column:f_116"`             // 尿蛋白定性
	F117        string `json:"f_117" gorm:"column:f_117"`             // 尿胆原定性
	F118        string `json:"f_118" gorm:"column:f_118"`             // 尿胆红素定性
	F119        string `json:"f_119" gorm:"column:f_119"`             // 尿红细胞定性
	F120        string `json:"f_120" gorm:"column:f_120"`             // 尿红细胞定量
	F121        string `json:"f_121" gorm:"column:f_121"`             // 尿白细胞定量
	F122        string `json:"f_122" gorm:"column:f_122"`             // 尿细菌定量
	F123        string `json:"f_123" gorm:"column:f_123"`             // 第一次
	F124        string `json:"f_124" gorm:"column:f_124"`             // 第二次
	F125        string `json:"f_125" gorm:"column:f_125"`             // 第三次
	F126        string `json:"f_126" gorm:"column:f_126"`             // 抗核抗体（1：10）
	F127        string `json:"f_127" gorm:"column:f_127"`             // 抗核抗体（1：20）
	F128        string `json:"f_128" gorm:"column:f_128"`             // 抗核抗体(1:32)
	F129        string `json:"f_129" gorm:"column:f_129"`             // 抗核抗体（1：40）
	F130        string `json:"f_130" gorm:"column:f_130"`             // 抗核抗体（1：80）
	F131        string `json:"f_131" gorm:"column:f_131"`             // 抗核抗体(1:100)
	F132        string `json:"f_132" gorm:"column:f_132"`             // 抗核抗体(1:160)
	F133        string `json:"f_133" gorm:"column:f_133"`             // 抗核抗体(1:320)
	F134        string `json:"f_134" gorm:"column:f_134"`             // 抗核抗体(1:640)
	F135        string `json:"f_135" gorm:"column:f_135"`             // 抗核抗体(1:1000)
	F136        string `json:"f_136" gorm:"column:f_136"`             // 抗核抗体(1:1280)
	F137        string `json:"f_137" gorm:"column:f_137"`             // 抗核抗体(1:2560)
	F138        string `json:"f_138" gorm:"column:f_138"`             // 抗核抗体(1:3200)
	F139        string `json:"f_139" gorm:"column:f_139"`             // 抗核抗体(1:5120)
	F140        string `json:"f_140" gorm:"column:f_140"`             // 抗核抗体(1:10000)
	F141        string `json:"f_141" gorm:"column:f_141"`             // 1：32000
	F142        string `json:"f_142" gorm:"column:f_142"`             // 1：100000
	F143        string `json:"f_143" gorm:"column:f_143"`             // 抗核抗体(1:20480)
	F144        string `json:"f_144" gorm:"column:f_144"`             // 抗核抗体（定性）
	F145        string `json:"f_145" gorm:"column:f_145"`             // 抗U1-nRNP抗体
	F146        string `json:"f_146" gorm:"column:f_146"`             // 抗U1-snRNP抗体
	F147        string `json:"f_147" gorm:"column:f_147"`             // 抗nRNP/Sm
	F148        string `json:"f_148" gorm:"column:f_148"`             // 抗sm
	F149        string `json:"f_149" gorm:"column:f_149"`             // 抗SSA
	F150        string `json:"f_150" gorm:"column:f_150"`             // 抗Ro-52
	F151        string `json:"f_151" gorm:"column:f_151"`             // 抗SSB
	F152        string `json:"f_152" gorm:"column:f_152"`             // 抗ScL-70
	F153        string `json:"f_153" gorm:"column:f_153"`             // 抗PM-Scl
	F154        string `json:"f_154" gorm:"column:f_154"`             // 抗Jo-1
	F155        string `json:"f_155" gorm:"column:f_155"`             // 抗着丝点蛋白B
	F156        string `json:"f_156" gorm:"column:f_156"`             // 抗增殖细胞核抗原
	F157        string `json:"f_157" gorm:"column:f_157"`             // 抗双链DNA抗体
	F158        string `json:"f_158" gorm:"column:f_158"`             // 抗核小体
	F159        string `json:"f_159" gorm:"column:f_159"`             // 抗组蛋白
	F160        string `json:"f_160" gorm:"column:f_160"`             // 抗核糖体P蛋白
	F161        string `json:"f_161" gorm:"column:f_161"`             // 自身免疫性肝病抗体检测
	F162        string `json:"f_162" gorm:"column:f_162"`             // ANA（IIF）
	F163        string `json:"f_163" gorm:"column:f_163"`             // 抗线粒体抗体
	F164        string `json:"f_164" gorm:"column:f_164"`             // 抗肝肾微粒体
	F165        string `json:"f_165" gorm:"column:f_165"`             // 抗肝抗原
	F166        string `json:"f_166" gorm:"column:f_166"`             // 抗平滑肌抗体
	F167        string `json:"f_167" gorm:"column:f_167"`             // 抗3E（BPO）
	F168        string `json:"f_168" gorm:"column:f_168"`             // 抗Sp100
	F169        string `json:"f_169" gorm:"column:f_169"`             // 抗PML
	F170        string `json:"f_170" gorm:"column:f_170"`             // 抗gp210
	F171        string `json:"f_171" gorm:"column:f_171"`             // 抗肝肾微粒体抗体
	F172        string `json:"f_172" gorm:"column:f_172"`             // 抗肝细胞溶质抗原I抗体
	F173        string `json:"f_173" gorm:"column:f_173"`             // 抗可溶性肝抗原/肝胰抗原抗体
	F174        string `json:"f_174" gorm:"column:f_174"`             // 抗线粒体M2
	F175        string `json:"f_175" gorm:"column:f_175"`             // pANCA
	F176        string `json:"f_176" gorm:"column:f_176"`             // cANCA
	F177        string `json:"f_177" gorm:"column:f_177"`             // 抗sm抗体
	F178        string `json:"f_178" gorm:"column:f_178"`             // 胃镜结果
	F179        string `json:"f_179" gorm:"column:f_179"`             // 肠镜结果
	F180        string `json:"f_180" gorm:"column:f_180"`             // 纤维扫描成功次数
	F181        string `json:"f_181" gorm:"column:f_181"`             // 脂肪衰减(db/m)
	F182        string `json:"f_182" gorm:"column:f_182"`             // 脂肪衰减四分位差(db/m)
	F183        string `json:"f_183" gorm:"column:f_183"`             // 硬度(Kpa)
	F184        string `json:"f_184" gorm:"column:f_184"`             // 硬度四分位差(Kpa)
	F185        string `json:"f_185" gorm:"column:f_185"`             // 纤维扫描总次数
	F186        string `json:"f_186" gorm:"column:f_186"`             // 电话
	F187        string `json:"f_187" gorm:"column:f_187"`             // 患者身高(cm)
	F188        string `json:"f_188" gorm:"column:f_188"`             // 患者体重(kg)
	F189        string `json:"f_189" gorm:"column:f_189"`             // 检测时长(s)
	F190        string `json:"f_190" gorm:"column:f_190"`             // 成功总次数
	F191        string `json:"f_191" gorm:"column:f_191"`             // B超诊断
	F192        string `json:"f_192" gorm:"column:f_192"`             // 肝脏 一期没存东西 后面实际存：	检查所见
	F193        string `json:"f_193" gorm:"column:f_193"`             // 胆囊相关疾病 一期没存东西 后面实际存：	肝脏
	F194        string `json:"f_194" gorm:"column:f_194"`             // 脾脏 一期没存东西 后面实际存：	胆囊
	F195        string `json:"f_195" gorm:"column:f_195"`             // 门静脉 一期没存东西 后面实际存：	胰腺
	F196        string `json:"f_196" gorm:"column:f_196"`             // 脾静脉 一期没存东西 后面实际存：	脾脏
	F197        string `json:"f_197" gorm:"column:f_197"`             // 腹腔积液 一期没存东西 后面实际存：	门静脉
	F198        string `json:"f_198" gorm:"column:f_198"`             // 肝门部肿大淋巴结大小 一期没存东西 后面实际存：	脾静脉
	F199        string `json:"f_199" gorm:"column:f_199"`             // 右侧颈动脉内中膜厚度（三次测量结果） 一期没存东西 后面实际存：	腹腔积液
	F200        string `json:"f_200" gorm:"column:f_200"`             // 甲状腺超声 一期没存东西 后面实际存：	右侧颈总动脉内-中膜
	F201        string `json:"f_201" gorm:"column:f_201"`             // 肝穿是否出血
	F202        string `json:"f_202" gorm:"column:f_202"`             // 肝穿出血处理措施
	F203        string `json:"f_203" gorm:"column:f_203"`             // 肝穿次数
	F204        int    `json:"f_204" gorm:"column:f_204"`             // 肝穿时间
	F205        string `json:"f_205" gorm:"column:f_205"`             // 肝穿病理号
	F206        string `json:"f_206" gorm:"column:f_206"`             // 肝穿病理结果
	F207        string `json:"f_207" gorm:"column:f_207"`             // G
	F208        string `json:"f_208" gorm:"column:f_208"`             // S
	F209        string `json:"f_209" gorm:"column:f_209"`             // 病理分期
	F210        string `json:"f_210" gorm:"column:f_210"`             // AIH评分
	F211        string `json:"f_211" gorm:"column:f_211"`             // CT/MRI
	F212        string `json:"f_212" gorm:"column:f_212"`             // UDCA
	F213        string `json:"f_213" gorm:"column:f_213"`             // UDCA备注
	F214        string `json:"f_214" gorm:"column:f_214"`             // 激素
	F215        string `json:"f_215" gorm:"column:f_215"`             // 非诺贝特
	F216        string `json:"f_216" gorm:"column:f_216"`             // 扶正化瘀/安络化纤
	F217        string `json:"f_217" gorm:"column:f_217"`             // 维生素E/维生素AD/钙
	F218        string `json:"f_218" gorm:"column:f_218"`             // 硫唑嘌呤（免疫制剂）
	F219        string `json:"f_219" gorm:"column:f_219"`             // 骁悉（免疫制剂）
	F220        string `json:"f_220" gorm:"column:f_220"`             // 利福昔明
	F221        string `json:"f_221" gorm:"column:f_221"`             // 清幽
	F222        string `json:"f_222" gorm:"column:f_222"`             // C13
	F223        string `json:"f_223" gorm:"column:f_223"`             // 他汀类用药
	F224        string `json:"f_224" gorm:"column:f_224"`             // 心得安/普萘洛尔（曲张，预防出血）
	F225        string `json:"f_225" gorm:"column:f_225"`             // 卡维地洛（曲张，预防出血）
	F226        string `json:"f_226" gorm:"column:f_226"`             // 其他用药
	F227        string `json:"f_227" gorm:"column:f_227"`             // 抗病毒药物
	F228        string `json:"f_228" gorm:"column:f_228"`             // 干扰素
	F229        string `json:"f_229" gorm:"column:f_229"`             // 利尿
	F230        string `json:"f_230" gorm:"column:f_230"`             // 乙肝E抗体定性
	F231        string `json:"f_231" gorm:"column:f_231"`             // 乙肝E抗原定性
	F232        string `json:"f_232" gorm:"column:f_232"`             // 乙肝表面抗体定性
	F233        string `json:"f_233" gorm:"column:f_233"`             // 乙肝表面抗原定性
	F234        string `json:"f_234" gorm:"column:f_234"`             // 乙肝核心抗体定性
	F235        string `json:"f_235" gorm:"column:f_235"`             // 乙肝前S1抗原定性
	F236        string `json:"f_236" gorm:"column:f_236"`             // 乙肝E抗体定量
	F237        string `json:"f_237" gorm:"column:f_237"`             // 乙肝E抗原定量
	F238        string `json:"f_238" gorm:"column:f_238"`             // 乙肝表面抗体定量
	F239        string `json:"f_239" gorm:"column:f_239"`             // 乙肝表面抗原定量
	F240        string `json:"f_240" gorm:"column:f_240"`             // 乙肝核心抗体定量
	F241        string `json:"f_241" gorm:"column:f_241"`             // 高敏乙肝DNA定量
	F242        string `json:"f_242" gorm:"column:f_242"`             // 乙肝病毒定量
	F243        string `json:"f_243" gorm:"column:f_243"`             // 高敏丙肝RNA定量
	F244        string `json:"f_244" gorm:"column:f_244"`             // 丙肝病毒定量
	F245        string `json:"f_245" gorm:"column:f_245"`             // 丙肝抗体定性
	F246        string `json:"f_246" gorm:"column:f_246"`             // 丙肝病毒基因分型
	F247        string `json:"f_247" gorm:"column:f_247"`             // 双肾
	F248        string `json:"f_248" gorm:"column:f_248"`             // 甲状腺
	F249        string `json:"f_249" gorm:"column:f_249"`             // 颈部

	IsConflict int    `json:"isConflict" gorm:"column:isConflict"`
	CreateTime string `json:"createTime" gorm:"column:createTime"` // 记录创建时间
	UpdateTime string `json:"updateTime" gorm:"column:updateTime"` // 记录最后更新时间
}

func (m *TCollect) TableName() string {
	return "t_collect"
}

func (m *TCollect) String() string {
	b, err := json.Marshal(*m)
	if err != nil {
		return fmt.Sprintf("%+v", *m)
	}
	var out bytes.Buffer
	err = json.Indent(&out, b, "", "    ")
	if err != nil {
		return fmt.Sprintf("%+v", *m)
	}
	return out.String()
}
