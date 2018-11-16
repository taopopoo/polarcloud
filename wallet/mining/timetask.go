package mining

import (
	"fmt"
)

//import (
//	"encoding/json"
//	"fmt"
//	//	"time"
//	//	"polarcloud/core/engine"
//	"polarcloud/core/utils"
//)

const (
	Task_class_buildBlock = "Task_class_buildBlock" //定时生成块
)

/*
	定时生成块
*/
func TaskBuildBlock(class, params string) {
	fmt.Printf("\n %c[1;40;32m%s%c[0m\n\n", 0x1B, "开始构建区块", 0x1B)
	go BuildBlock()
}
