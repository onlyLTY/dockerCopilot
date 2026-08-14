package svc

// TryStartContainerDelete reserves a container for a batch delete task.
func (ctx *ServiceContext) TryStartContainerDelete(containerID, taskID string) bool {
	if ctx == nil || containerID == "" {
		return false
	}
	ctx.updateMu.Lock()
	defer ctx.updateMu.Unlock()
	ctx.containerOpsMu.Lock()
	defer ctx.containerOpsMu.Unlock()
	if ctx.containerDeletes == nil {
		ctx.containerDeletes = make(map[string]string)
	}
	if ctx.updatingContainers != nil {
		if _, updating := ctx.updatingContainers[containerID]; updating {
			return false
		}
	}
	if _, exists := ctx.containerDeletes[containerID]; exists {
		return false
	}
	ctx.containerDeletes[containerID] = taskID
	return true
}

func (ctx *ServiceContext) FinishContainerDelete(containerID, taskID string) {
	if ctx == nil || containerID == "" {
		return
	}
	ctx.updateMu.Lock()
	defer ctx.updateMu.Unlock()
	ctx.containerOpsMu.Lock()
	defer ctx.containerOpsMu.Unlock()
	if current, ok := ctx.containerDeletes[containerID]; ok && current == taskID {
		delete(ctx.containerDeletes, containerID)
	}
}
