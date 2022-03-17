package dice

func RegisterBuiltinExtDnd5e(self *Dice) {
	theExt := &ExtInfo{
		Name:       "dnd5e", // 扩展的名称，需要用于开启和关闭指令中，写简短点
		Version:    "1.0.0",
		Brief:      "不要看了，还没开始。咕咕咕",
		AutoActive: false, // 是否自动开启
		OnCommandReceived: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) {
			//p := getPlayerInfoBySender(session, msg)
			//p.TempValueAlias = &ac.Alias;
		},
		GetDescText: func(i *ExtInfo) string {
			text := "> " + i.Brief + "\n" + "提供命令:\n"
			for _, i := range i.CmdMap {
				brief := i.Brief
				if brief != "" {
					brief = " // " + brief
				}
				text += "." + i.Name + brief + "\n"
			}
			return text
		},
		CmdMap: CmdMapCls{},
	}

	self.RegisterExtension(theExt)
}
