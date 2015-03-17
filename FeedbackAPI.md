# 意见反馈接口使用规范 #

```
请求地址：http://<addr>/mail

参数：
    subject 主题
    content 内容    
    charset 编码（可选）
    code    客户端生成的随机数（所有用户的客户端24小时内不能重复）
    verify  校验码

返回值：    
    成功: OK 状态码：200
    校验错误：invalid request 状态码：400
    其它错误：<错误原因> 状态码：400
        
校验码计算方法:
    hmac + sha1 （subject+content+code) hex     
    
    PHP 示例：
        hash_hmac ("sha1", $subject.$content.$code, $hmac_key)
    
    Python 示例：
        hmac.new(<hmac_key>, subject + content + code), hashlib.sha1).hexdigest()
        
    Golang 示例:
        h := hmac.New(sha1.New, []byte(hmac_key))
        h.Write([]byte(subject))
        h.Write([]byte(content))
        h.Write([]byte(code))
        fmt.Sprintf("%x", h.Sum(nil))
        
    Java 示例：
        不知道怎么写...
        
    注意：    
        subject和content必须是urlencode之前的
        hmac-key 请联系 QQ:297280699 Qi 索取    
```

---

### 发布规范 ###

  * subject统一为“**意见反馈：校巴定位**”，不用显示给用户。
  * 有**联系方式**和**反馈内容\*两个输入框。**联系方式**为非必填，在placeholder里提示“电话/邮箱/QQ/微博”。
  * 提交时将**反馈内容\*、**联系信息**和**平台的一些信息（比如操作系统、应用版本、浏览器什么的）**连起来作为content提交。


---

### 邮箱更改 ###

http://

&lt;addr&gt;

/set?email=<新的email地址>&password=<新的email密码>&smtp\_server=smtp.gmail.com:587&auth=<上一次更改邮箱时设置的认证码>&new\_auth=<下一次更改使用的认证码>