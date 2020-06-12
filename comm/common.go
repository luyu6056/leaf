package comm

//最大连接数量
const TCP_MAX_CONN_NUM = 2000

// 包长度字段
const TCP_MSG_LEN = 4
const TCP_MSG_LEN_SLG = 2

// 内部服务器间的数据包通讯最大长度
const TCP_MSG_LEN_IN_SVR = 102400

// 服务器同客户端数据包通讯最大长度
const TCP_MSG_LEN_CLIENT = 109600

// 服务器同客户端数据包通讯最大长度
const TCP_MSG_LEN_CLIENT_LS = 4096

// 每日刷新时间, 早上5点
const EveryDay_FlushTime = 5
