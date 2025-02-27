package pixiv

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/RicheyJang/PaimengBot/manager"
	"github.com/RicheyJang/PaimengBot/utils"
	"github.com/RicheyJang/PaimengBot/utils/consts"

	zero "github.com/wdvxdr1123/ZeroBot"
)

var info = manager.PluginInfo{
	Name: "好康的",
	Usage: `用法：
	美图/涩图 [Tag]* [数量num]?：num(默认1张)张随机Pixiv美图，来自经过筛选的图库
示例：
	美图 胡桃 2：丢给你两张精选胡桃的美(se)图~
	来两张胡桃的涩图：等同于上一条
另外，高级用法询问管理员哦~[dog]`,
	SuperUsage: `特别用法：(在私聊中)
	色图r [Tag]* [数量num]?：你懂得
config-plugin配置项：
	pixiv.timeout： 下载图片超时时长，至少为1s；越长下载成功率越高、等待时间越长
	pixiv.proxy： Pixiv反代网站，默认为i.pixiv.re，令外可选i.pixiv.cat
	pixiv.scale：从各个图库取图的比例，导入Omega图库后，将pixiv.scale.omega设为非0值才可使其生效
	pixiv.r18：是否允许18+，设为false则开启强力不可以涩涩模式
	pixiv.groupr18：是(true)否(false)允许在群聊中发送18+，默认为false
	pixiv.omega.setu：在请求非R18图片时，是(true)否(false)从Omega图库中拿取nsfw=1(setu)的图片
另外，Omega图库是指从https://github.com/Ailitonia/omega-miya/raw/master/archive_data/db_pixiv.7z手动导入数据库`,
	Classify: "好康的",
}
var proxy *manager.PluginProxy

type PictureInfo struct {
	Title string // 标题

	// 下载所需
	URL string // 图片链接
	PID int64  // 下载图片时要么有URL；要么有PID及P
	P   int    // 分P

	// 描述所需
	Tags   []string // 标签
	Author string   // 作者
	UID    int64    // 作者UID

	Src string // 无需填写，来源图库
}
type PictureGetter func(tags []string, num int, isR18 bool) []PictureInfo

var ( // 若有新的图库加入，修改以下两个Map即可，会自动适配
	getterMap = map[string]PictureGetter{ // 各个图库的取图函数映射
		"lolicon": getPicturesFromLolicon,
		"omega":   getPicturesFromOmega,
	}
	getterScale = map[string]int{ // 从各个图库取图的初始比例
		"lolicon": 5,
		"omega":   0,
	}
)

func init() {
	proxy = manager.RegisterPlugin(info)
	if proxy == nil {
		return
	}
	proxy.OnCommands([]string{"美图r", "涩图r", "色图r", "瑟图r"}).SetBlock(true).SecondPriority().Handle(getPictures)
	proxy.OnCommands([]string{"美图", "涩图", "色图", "瑟图"}).SetBlock(true).ThirdPriority().Handle(getPictures)
	proxy.OnRegex(`^来?([\d一两二三四五六七八九十]*)[张页点份发](.*)[色涩美瑟]图([rR]?)$`).SetBlock(true).SetPriority(4).Handle(getPicturesWithRegex)
	proxy.AddConfig("omega.setu", false) // 在请求非R18图片时，是否从Omega图库中拿取nsfw=1(setu)的图片
	proxy.AddAPIConfig(consts.APIOfHibiAPIKey, "api.obfs.dev")
	proxy.AddConfig("timeout", "10s") // 下载图片超时时长 格式要求time.ParseDuration 至少为1s
	proxy.AddConfig("proxy", "i.pixiv.re")
	proxy.AddConfig("groupr18", false)
	proxy.AddConfig("r18", true)
	for k, v := range getterScale { // 各个图库取图比例配置
		proxy.AddConfig(fmt.Sprintf("scale.%s", k), v)
	}
}

// 消息处理函数 -----

func getPictures(ctx *zero.Ctx) {
	// 命令
	isR := false
	cmd := utils.GetCommand(ctx)
	if strings.HasSuffix(cmd, "r") || strings.HasSuffix(cmd, "R") {
		if !utils.IsMessagePrimary(ctx) && !proxy.GetConfigBool("groupr18") {
			ctx.Send("滚滚滚")
			return
		}
		isR = true
	}
	// 参数
	arg := strings.TrimSpace(utils.GetArgs(ctx))
	args := strings.Split(arg, " ")
	num := getCmdNum(args[len(args)-1])
	if num > 1 {
		args = args[:len(args)-1]
	}
	// 发图
	newDownloader(args, num, isR).send(ctx)
}

func getPicturesWithRegex(ctx *zero.Ctx) {
	subs := utils.GetRegexpMatched(ctx)
	if len(subs) <= 3 { // 正则出错
		ctx.Send("？")
		return
	}
	num := getCmdNum(subs[1])
	subs[2] = strings.ReplaceAll(subs[2], "的", " ")
	tags := strings.Split(subs[2], " ")
	tags = utils.MergeStringSlices(tags)
	isR := false
	if len(subs[3]) > 0 {
		if !utils.IsMessagePrimary(ctx) && !proxy.GetConfigBool("groupr18") {
			ctx.Send("滚滚滚")
			return
		}
		isR = true
	}
	// 发图
	newDownloader(tags, num, isR).send(ctx)
}

func getCmdNum(num string) int {
	if r, ok := chineseNumToInt[num]; ok {
		return r
	}
	r, err := strconv.Atoi(num)
	if err != nil || r <= 0 {
		return 1
	}
	return r
}

var chineseNumToInt = map[string]int{
	"一": 1,
	"两": 2,
	"二": 2,
	"三": 3,
	"四": 4,
	"五": 5,
	"六": 6,
	"七": 7,
	"八": 8,
	"九": 9,
	"十": 10,
}
