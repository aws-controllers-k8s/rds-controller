    if err == nil {
		_ = resp
        err = ackrequeue.Needed(fmt.Errorf("wait for DBInstance deletion"))
    }