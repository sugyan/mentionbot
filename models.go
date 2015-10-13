package mentionbot

// IDs is id list
type IDs []string

// CursoredIDs is users list with cursor info
type CursoredIDs map[string]interface{}

// NextCursorStr returns next cursor
func (ci CursoredIDs) NextCursorStr() string {
	return ci["next_cursor_str"].(string)
}

// PreviousCursorStr returns previous cursor
func (ci CursoredIDs) PreviousCursorStr() string {
	return ci["previous_cursor_str"].(string)
}

// IDs returns users list
func (ci CursoredIDs) IDs() IDs {
	results := ci["ids"].([]interface{})
	ids := make([]string, len(results))
	for i, value := range results {
		ids[i] = value.(string)
	}
	return ids
}
