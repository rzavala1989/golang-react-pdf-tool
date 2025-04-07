package helpers

import (
	"fmt"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ExtractFormFields returns a map of field names to their current values from the provided PDF.
func ExtractFormFields(filePath string) (map[string]string, error) {
	ctx, err := api.ReadContextFile(filePath)
	if err != nil {
		return nil, err
	}

	root := ctx.XRefTable.RootDict
	acroFormRef, found := root.Find("AcroForm")
	if !found {
		return nil, fmt.Errorf("no AcroForm found in PDF")
	}

	acroFormDict, err := ctx.DereferenceDict(acroFormRef)
	if err != nil || acroFormDict == nil {
		return nil, fmt.Errorf("could not dereference AcroForm dictionary")
	}

	fieldsRef, found := acroFormDict.Find("Fields")
	if !found {
		return nil, fmt.Errorf("AcroForm has no Fields entry")
	}

	fields, err := ctx.DereferenceArray(fieldsRef)
	if err != nil {
		return nil, fmt.Errorf("could not dereference Fields array")
	}

	fieldMap := make(map[string]string)
	for _, fieldRef := range fields {
		fieldDict, err := ctx.DereferenceDict(fieldRef)
		if err != nil || fieldDict == nil {
			continue
		}
		collectFields(ctx, fieldMap, fieldDict, "")
	}

	return fieldMap, nil
}

// collectFields walks PDF field dictionaries recursively and extracts field names + values.
func collectFields(ctx *model.Context, fieldMap map[string]string, field types.Dict, parent string) {
	tObj := field["T"]
	if tObj == nil {
		return
	}

	dObj, err := ctx.Dereference(tObj)
	if err != nil {
		return
	}
	tStr := dObj.String()
	fullName := tStr
	if parent != "" {
		fullName = parent + "." + tStr
	}

	vObj, found := field.Find("V")
	valStr := ""
	if found {
		derefVal, err := ctx.Dereference(vObj)
		if err == nil && derefVal != nil {
			valStr = derefVal.String()
		}
	}

	fieldMap[fullName] = valStr

	if kidsObj, found := field.Find("Kids"); found {
		kidsArray, err := ctx.DereferenceArray(kidsObj)
		if err != nil {
			return
		}
		for _, kid := range kidsArray {
			kidDict, err := ctx.DereferenceDict(kid)
			if err != nil || kidDict == nil {
				continue
			}
			collectFields(ctx, fieldMap, kidDict, fullName)
		}
	}
}

// GenerateFDF creates minimal FDF content for pdfcpu.FillFormFile using field name/value pairs.
func GenerateFDF(data map[string]string) string {
	fdfHeader := `%FDF-1.2
%âãÏÓ
1 0 obj
<<
/FDF <<
/Fields [
`
	fdfFooter := `]
>>
>>
endobj
trailer
<<
/Root 1 0 R
>>
%%EOF
`
	fieldsContent := ""
	for k, v := range data {
		// You may want to escape `()` or special characters here in a production-grade app.
		fieldsContent += fmt.Sprintf("<< /T (%s) /V (%s) >>\n", k, v)
	}
	return fdfHeader + fieldsContent + fdfFooter
}
