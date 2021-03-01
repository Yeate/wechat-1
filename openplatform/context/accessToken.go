//Package context 开放平台相关context
package context

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/silenceper/wechat/v2/util"
)

const (
	componentAccessTokenURL = "https://api.weixin.qq.com/cgi-bin/component/api_component_token"
	getPreCodeURL           = "https://api.weixin.qq.com/cgi-bin/component/api_create_preauthcode?component_access_token=%s"
	queryAuthURL            = "https://api.weixin.qq.com/cgi-bin/component/api_query_auth?component_access_token=%s"
	refreshTokenURL         = "https://api.weixin.qq.com/cgi-bin/component/api_authorizer_token?component_access_token=%s"
	getComponentInfoURL     = "https://api.weixin.qq.com/cgi-bin/component/api_get_authorizer_info?component_access_token=%s"
	componentLoginURL       = "https://mp.weixin.qq.com/cgi-bin/componentloginpage?component_appid=%s&pre_auth_code=%s&redirect_uri=%s&auth_type=%d&biz_appid=%s"
	bindComponentURL        = "https://mp.weixin.qq.com/safe/bindcomponent?action=bindcomponent&auth_type=%d&no_scan=1&component_appid=%s&pre_auth_code=%s&redirect_uri=%s&biz_appid=%s#wechat_redirect"
	//TODO 获取授权方选项信息
	//getComponentConfigURL = "https://api.weixin.qq.com/cgi-bin/component/api_get_authorizer_option?component_access_token=%s"
	//TODO 获取已授权的账号信息
	//getuthorizerListURL = "POST https://api.weixin.qq.com/cgi-bin/component/api_get_authorizer_list?component_access_token=%s"
	getCodeTemplate = "https://api.weixin.qq.com/wxa/gettemplatelist?access_token=%s"
	getFastRegisterAuthURL     = "https://mp.weixin.qq.com/cgi-bin/fastregisterauth?appid=%s&component_appid=%s&copy_wx_verify=1&redirect_uri=%s"

)

// ComponentAccessToken 第三方平台
type ComponentAccessToken struct {
	AccessToken string `json:"component_access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// GetComponentAccessToken 获取 ComponentAccessToken
func (ctx *Context) GetComponentAccessToken() (string, error) {
	accessTokenCacheKey := fmt.Sprintf("component_access_token_%s", ctx.AppID)
	val := ctx.Cache.Get(accessTokenCacheKey)
	if val == nil {
		return "", fmt.Errorf("cann't get component access token")
	}
	return val.(string), nil
}

// SetComponentAccessToken 通过component_verify_ticket 获取 ComponentAccessToken
func (ctx *Context) SetComponentAccessToken(verifyTicket string) (*ComponentAccessToken, error) {
	body := map[string]string{
		"component_appid":         ctx.AppID,
		"component_appsecret":     ctx.AppSecret,
		"component_verify_ticket": verifyTicket,
	}
	respBody, err := util.PostJSON(componentAccessTokenURL, body)
	if err != nil {
		return nil, err
	}

	at := &ComponentAccessToken{}
	fmt.Println(string(respBody))
	if err := json.Unmarshal(respBody, at); err != nil {
		return nil, err
	}

	accessTokenCacheKey := fmt.Sprintf("component_access_token_%s", ctx.AppID)
	expires := at.ExpiresIn - 1500
	if err := ctx.Cache.Set(accessTokenCacheKey, at.AccessToken, time.Duration(expires)*time.Second); err != nil {
		fmt.Println(err)
		return nil, nil
	}
	return at, nil
}

// GetPreCode 获取预授权码
func (ctx *Context) GetPreCode() (string, error) {
	cat, err := ctx.GetComponentAccessToken()
	if err != nil {
		return "", err
	}
	req := map[string]string{
		"component_appid": ctx.AppID,
	}
	uri := fmt.Sprintf(getPreCodeURL, cat)
	body, err := util.PostJSON(uri, req)
	if err != nil {
		return "", err
	}

	var ret struct {
		PreCode string `json:"pre_auth_code"`
	}
	if err := json.Unmarshal(body, &ret); err != nil {
		return "", err
	}

	return ret.PreCode, nil
}

// GetComponentLoginPage 获取第三方公众号授权链接(扫码授权)
func (ctx *Context) GetComponentLoginPage(redirectURI string, authType int, bizAppID string) (string, error) {
	code, err := ctx.GetPreCode()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(componentLoginURL, ctx.AppID, code, url.QueryEscape(redirectURI), authType, bizAppID), nil
}

// GetBindComponentURL 获取第三方公众号授权链接(链接跳转，适用移动端)
func (ctx *Context) GetBindComponentURL(redirectURI string, authType int, bizAppID string) (string, error) {
	code, err := ctx.GetPreCode()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(bindComponentURL, authType, ctx.AppID, code, url.QueryEscape(redirectURI), bizAppID), nil
}

func (ctx *Context) GetFastRegisterAuth(gzhAppId string,redirectUrl string)(url string){
	url = fmt.Sprintf(getFastRegisterAuthURL,gzhAppId, ctx.AppID,redirectUrl)
	return
}

// ID 微信返回接口中各种类型字段
type ID struct {
	ID int `json:"id"`
}

// AuthBaseInfo 授权的基本信息
type AuthBaseInfo struct {
	AuthrAccessToken
	FuncInfo []AuthFuncInfo `json:"func_info"`
}

// AuthFuncInfo 授权的接口内容
type AuthFuncInfo struct {
	FuncscopeCategory ID `json:"funcscope_category"`
}

// AuthrAccessToken 授权方AccessToken
type AuthrAccessToken struct {
	Appid        string `json:"authorizer_appid"`
	AccessToken  string `json:"authorizer_access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"authorizer_refresh_token"`
}

// QueryAuthCode 使用授权码换取公众号或小程序的接口调用凭据和授权信息
func (ctx *Context) QueryAuthCode(authCode string) (*AuthBaseInfo, error) {
	cat, err := ctx.GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	req := map[string]string{
		"component_appid":    ctx.AppID,
		"authorization_code": authCode,
	}
	uri := fmt.Sprintf(queryAuthURL, cat)
	body, err := util.PostJSON(uri, req)
	if err != nil {
		return nil, err
	}

	var ret struct {
		util.CommonError
		Info *AuthBaseInfo `json:"authorization_info"`
	}

	if err := json.Unmarshal(body, &ret); err != nil {
		return nil, err
	}
	if ret.ErrCode != 0 {
		err = fmt.Errorf("QueryAuthCode error : errcode=%v , errmsg=%v", ret.ErrCode, ret.ErrMsg)
		return nil, err
	}
	return ret.Info, nil
}

// RefreshAuthrToken 获取（刷新）授权公众号或小程序的接口调用凭据（令牌）
func (ctx *Context) RefreshAuthrToken(appid, refreshToken string) (*AuthrAccessToken, error) {
	cat, err := ctx.GetComponentAccessToken()
	if err != nil {
		return nil, err
	}

	req := map[string]string{
		"component_appid":          ctx.AppID,
		"authorizer_appid":         appid,
		"authorizer_refresh_token": refreshToken,
	}
	uri := fmt.Sprintf(refreshTokenURL, cat)
	body, err := util.PostJSON(uri, req)
	if err != nil {
		return nil, err
	}

	ret := &AuthrAccessToken{}
	if err := json.Unmarshal(body, ret); err != nil {
		return nil, err
	}

	authrTokenKey := "authorizer_access_token_" + appid
	if err := ctx.Cache.Set(authrTokenKey, ret.AccessToken, time.Minute*80); err != nil {
		return nil, err
	}
	return ret, nil
}

// GetAuthrAccessToken 获取授权方AccessToken
func (ctx *Context) GetAuthrAccessToken(appid string) (string, error) {
	authrTokenKey := "authorizer_access_token_" + appid
	val := ctx.Cache.Get(authrTokenKey)
	if val == nil {
		return "", fmt.Errorf("cannot get authorizer %s access token", appid)
	}
	return val.(string), nil
}

// AuthorizerInfo 授权方详细信息
type AuthorizerInfo struct {
	NickName        string `json:"nick_name"`
	HeadImg         string `json:"head_img"`
	ServiceTypeInfo ID     `json:"service_type_info"`
	VerifyTypeInfo  ID     `json:"verify_type_info"`
	UserName        string `json:"user_name"`
	PrincipalName   string `json:"principal_name"`
	BusinessInfo    struct {
		OpenStore string `json:"open_store"`
		OpenScan  string `json:"open_scan"`
		OpenPay   string `json:"open_pay"`
		OpenCard  string `json:"open_card"`
		OpenShake string `json:"open_shake"`
	}
	Alias           string          `json:"alias"`
	QrcodeURL       string          `json:"qrcode_url"`
	MiniprogramInfo MiniProgramInfo `json:"MiniProgramInfo"`
}

type MiniProgramInfo struct {
	Network     MiniProgramNetwork    `json:"network"`
	Categories  []MiniProgramCategory `json:"categories"`
	VisitStatus int64                 `json:"visit_status"`
	Exists      bool                  `json:"exists"`
}
type MiniProgramNetwork struct {
	RequestDomain   []string `json:"RequestDomain"`
	WsRequestDomain []string `json:"WsRequestDomain"`
	UploadDomain    []string `json:"UploadDomain"`
	DownloadDomain  []string `json:"DownloadDomain"`
}
type MiniProgramCategory struct {
	First  string `json:"first"`
	Second string `json:"second"`
}

// GetAuthrInfo 获取授权方的帐号基本信息
func (ctx *Context) GetAuthrInfo(appid string) (*AuthorizerInfo, *AuthBaseInfo, error) {
	cat, err := ctx.GetComponentAccessToken()
	if err != nil {
		return nil, nil, err
	}

	req := map[string]string{
		"component_appid":  ctx.AppID,
		"authorizer_appid": appid,
	}

	uri := fmt.Sprintf(getComponentInfoURL, cat)
	body, err := util.PostJSON(uri, req)
	if err != nil {
		return nil, nil, err
	}

	var ret struct {
		AuthorizerInfo    *AuthorizerInfo `json:"authorizer_info"`
		AuthorizationInfo *AuthBaseInfo   `json:"authorization_info"`
	}
	if err := json.Unmarshal(body, &ret); err != nil {
		return nil, nil, err
	}
	retMap := make(map[string]map[string]interface{})

	if err := json.Unmarshal(body, &retMap); err != nil {
		return nil, nil, err
	}

	if _, ok := retMap["authorizer_info"]["MiniProgramInfo"]; ok {
		ret.AuthorizerInfo.MiniprogramInfo.Exists = ok
	}
	return ret.AuthorizerInfo, ret.AuthorizationInfo, nil
}

//TemplateList 代码模板列表
type TemplateList struct {
	CreateTime  int64  `json:"create_time"`
	UserVersion string `json:"user_version"`
	UserDesc    string `json:"user_desc"`
	TemplateId  int64  `json:"template_id"`
}

// getCodeTemplate 获取代码模板列表
func (ctx *Context) GetCodeTemplate() (templateList []*TemplateList, err error) {
	cat, err := ctx.GetComponentAccessToken()
	if err != nil {
		return
	}
	var ret struct {
		util.CommonError
		TemplateList []*TemplateList `json:"template_list"`
	}
	uri := fmt.Sprintf(getCodeTemplate, cat)
	data, err := util.HTTPGet(uri)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	if ret.ErrCode != 0 {
		err = fmt.Errorf("GetCodeTemplate error : errcode=%v , errmsg=%v", ret.ErrCode, ret.ErrMsg)
		return nil, err
	}
	templateList = ret.TemplateList
	return
}
