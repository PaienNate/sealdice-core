package v120

func CreateFakeCtx() *MsgContext {
	return &MsgContext{
		Dice: &Dice{
			DB: BoltDBInit("./data/default/data.bdb"),
		},
	}
}
