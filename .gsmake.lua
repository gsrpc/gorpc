name "github.com/gsrpc/gorpc"

plugin "github.com/gsmake/golang"
plugin "github.com/gsmake/gsrpc"


properties.golang = {
    dependencies = {
        { name = "github.com/gsdocker/gsos"     };
        { name = "github.com/gsdocker/gserrors" };
        { name = "github.com/gsdocker/gsconfig" };
        { name = "github.com/gsdocker/gslogger" };
        { name = "github.com/gorilla/websocket" };
    };

    tests = { "test" };
}

properties.gsrpc = {
    lang = "golang";
    dependencies = {
        { name = "github.com/gsrpc/gslang" };
        { name = "github.com/gsrpc/gsrpc" };
    }
}
