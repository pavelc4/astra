package terabox

import "regexp"

const (
	PlatformName      = "terabox"
	AdminAjaxEndpoint = "https://teradownloaderz.com/wp-admin/admin-ajax.php"
)

var (
	NonceRegex = regexp.MustCompile(`"nonce":"(.*?)"`)
	URLRegex   = regexp.MustCompile(`https?://(?:www\.)?(?:[a-zA-Z0-9-]+\.)?(?:terabox|1024tera|teraboxapp|teraboxlink|freeterabox)\.(?:com|app|link)/`)
)
