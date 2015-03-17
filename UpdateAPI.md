# 版本更新及公告获取接口 #

```
请求地址：http://www.100steps.net:8003/checkUpdate?platform=<android/ios/wp>

返回一个json，如下：
{
    Announce: {
        Caption: "今天是芥末日",
        Text: "哦！",
        CreatedAt: 1356074957
    },
    Version: {
        Version: "1.0",
        Desc: "囧囧囧囧囧囧",
        Url: "http://www.100steps.net",
        Time: 1356074980
    }
}
```