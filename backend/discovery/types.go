package discovery

type Route struct {
    Method      string `json:"method"`      
    Path        string `json:"path"`        
    HandlerName string `json:"handlerName"`  
    HandlerCode string `json:"handlerCode"`  
    Schema      Schema `json:"schema"`      
    File        string `json:"file"`         
    HandlerFile string `json:"handlerFile"`  
}

type Schema struct {
    BodyFields  []string `json:"bodyFields"`  
    PathParams  []string `json:"pathParams"`  
    QueryParams []string `json:"queryParams"` 
    ZodFields   []ZodField `json:"zodFields"` 
}

type ZodField struct {
    Name    string `json:"name"`    
    ZodType string `json:"zodType"` 
}
type Symbol struct {
    Name string 
    Code string 
    File string 
}
type MountPoint struct {
    Prefix     string
    RouterName string
    MountFn    string 
    File       string
}
type SymbolTable map[string]Symbol