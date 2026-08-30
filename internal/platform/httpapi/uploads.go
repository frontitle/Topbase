package httpapi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/topbase/topbase/internal/core"
	"github.com/xuri/excelize/v2"
)

const maxWorkbookSize = 20 << 20

func safeColumnName(value string, index int, used map[string]bool) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fmt.Sprintf("字段%d", index+1)
	}
	base, name := value, value
	for n := 2; used[name]; n++ {
		name = fmt.Sprintf("%s_%d", base, n)
	}
	used[name] = true
	return name
}

func (s *server) uploadWorkbook(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxWorkbookSize)
	if err := r.ParseMultipartForm(maxWorkbookSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Excel 文件不能超过 20 MB"})
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请选择 Excel 文件"})
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".xlsx" && ext != ".xlsm" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "目前支持 .xlsx 或 .xlsm 文件"})
		return
	}
	content, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "读取上传文件失败"})
		return
	}
	book, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无法解析 Excel 文件：" + err.Error()})
		return
	}
	defer book.Close()
	sheet := strings.TrimSpace(r.FormValue("sheet"))
	if sheet == "" {
		sheets := book.GetSheetList()
		if len(sheets) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Excel 中没有可导入的工作表"})
			return
		}
		sheet = sheets[0]
	}
	rows, err := book.GetRows(sheet)
	if err != nil || len(rows) < 2 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请选择至少包含表头和一行数据的工作表"})
		return
	}
	used := map[string]bool{}
	columns := make([]core.Column, len(rows[0]))
	for i, value := range rows[0] {
		columns[i] = core.Column{Name: safeColumnName(value, i, used), DataType: "text", Nullable: true}
	}
	data := make([][]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		if len(data) >= 100000 {
			break
		}
		values := make([]string, len(columns))
		copy(values, row)
		empty := true
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				empty = false
				break
			}
		}
		if !empty {
			data = append(data, values)
		}
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	}
	item := core.UploadedTable{ID: core.NewID("upl"), Name: name, FileName: header.Filename, SheetName: sheet, Columns: columns, RowCount: len(data), CreatedAt: time.Now().UTC()}
	if err := s.store.SaveUploadedTable(item, data); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存本地数据集失败：" + err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *server) listUploadedTables(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	items, err := s.store.ListUploadedTables()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, items)
}
func (s *server) deleteUploadedTable(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if err := s.store.DeleteUploadedTable(r.PathValue("id")); err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
