package mlr

// func TestMLR(t *testing.T) {
// 	tempFile := Value(os.CreateTemp("", "temp.csv"))
// 	defer (func () { os.Remove(tempFile.Name()) })()
// 	original := []byte(`| Item | UnitPrice | Quantity | Total |
// | --- | --- | --- | --- |
// | Apple | 1.5 | 12 | 0 |
// | Banana | 2.0 | 5 | 0 |
// | Orange | 1.2 | 8 | 0 |
// `)
// 	script := "$Total = $UnitPrice * $Quantity"
// 	expected := []byte(`| Item | UnitPrice | Quantity | Total |
// | --- | --- | --- | --- |
// | Apple | 1.5 | 12 | 18 |
// | Banana | 2.0 | 5 | 10 |
// | Orange | 1.2 | 8 | 9.6 |
// `)
// 	V0(os.WriteFile(tempFile, original, 0644))
// 	mlrMDInplacePut(tempFile.Name, script)
// 	result := V(os.ReadFile(tempFilePath))
// 	if !bytes.Equal(expected, result) {
// 		t.Fatalf("MLR test failed:\n%s", diff.LineDiff(string(expected), string(result)))
// 	}
// }
