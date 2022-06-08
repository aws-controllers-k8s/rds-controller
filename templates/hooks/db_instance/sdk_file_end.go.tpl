{{ $CRD := .CRD }}
{{ $SDKAPI := .SDKAPI }}

{{ range $operationName := MakeSlice "RestoreDBInstanceFromDBSnapshot" "CreateDBInstanceReadReplica" }}

{{- $operation := (index $SDKAPI.API.Operations $operationName)}}

{{- $inputRef := $operation.InputRef }}
{{- $inputShapeName := $inputRef.ShapeName }}

{{- $outputRef := $operation.OutputRef }}
{{- $outputShapeName := $outputRef.ShapeName }}

// new{{ $inputShapeName }} returns a {{ $inputShapeName }} object 
// with each the field set by the corresponding configuration's fields.
func (rm *resourceManager) new{{ $inputShapeName }}(
    r *resource,
) *svcsdk.{{ $inputShapeName }} {
    res := &svcsdk.{{ $inputShapeName }}{}

{{ GoCodeSetSDKForStruct $CRD "" "res" $inputRef "" "r.ko.Spec" 1 }}

    return res
}

// setResourceFrom{{ $outputShapeName }} sets a resource {{ $outputShapeName }} type
// given the SDK type.
func (rm *resourceManager) setResourceFrom{{ $outputShapeName }}(
    r *resource,
    resp *svcsdk.{{ $outputShapeName }},
) {
{{ GoCodeSetCreateOutput $CRD "resp" "r.ko" 1 }}
}

{{- end }}
