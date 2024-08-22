package directory

type Directory struct {
	Id        uint32       `json:"id"`
	ParentId  uint32       `json:"parentId"`
	Name      string       `json:"name"`
	Accept    string       `json:"accept"`
	MaxSize   uint32       `json:"maxSize"`
	CreatedAt int64        `json:"createdAt"`
	UpdatedAt int64        `json:"updatedAt"`
	Children  []*Directory `json:"Children"`
}

// ID 获取菜单树ID
func (m *Directory) ID() uint32 {
	return m.Id
}

// Parent 获取父ID
func (m *Directory) Parent() uint32 {
	return m.ParentId
}

// AppendChildren 添加子节点
func (m *Directory) AppendChildren(child *Directory) {
	m.Children = append(m.Children, child)
}

// ChildrenNode 获取子节点
func (m *Directory) ChildrenNode() []*Directory {
	var list []*Directory
	for _, item := range m.Children {
		list = append(list, item)
	}
	return list
}
