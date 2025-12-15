package config

import (
	"github.com/pos/api-gateway/utils"
)

var TenantServiceURL = utils.GetEnv("TENANT_SERVICE_URL", "http://localhost:8084")
var ProductServiceURL = utils.GetEnv("PRODUCT_SERVICE_URL", "http://localhost:8086")
var AuthServiceURL = utils.GetEnv("AUTH_SERVICE_URL", "http://localhost:8082")
var UserServiceURL = utils.GetEnv("USER_SERVICE_URL", "http://localhost:8083")
var OrderServiceURL = utils.GetEnv("ORDER_SERVICE_URL", "http://localhost:8087")
var NotificationServiceURL = utils.GetEnv("NOTIFICATION_SERVICE_URL", "http://localhost:8088")
