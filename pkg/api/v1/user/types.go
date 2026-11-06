package user

type UserInfoReq struct {
	UID  uint   `form:"uid"`
	Name string `form:"name"`
}

type UserInfoData struct {
	UID       uint   `json:"uid,omitempty"`
	Name      string `json:"user_name,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Email     string `json:"email,omitempty"`
}

type DummyReq struct {
	Id    int  `json:"id" binding:"required"`
	Panic bool `json:"panic"`
}

type DummyData struct {
	Echo int `json:"echo"`
}
