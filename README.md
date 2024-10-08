# OpenTelemetry 데모

## 사용법

### Windows

```
.\bin\demo.exe [옵션]
```

옵션 리스트

+ -up [all|서비스명]
    + docker compose up
+ -down [all|서비스명]
    + docker compose down
+ -stop [all|서비스명]
    + docker compose stop
+ -jar
    + 자동계측 java application run
+ -manual
    + 수동계측 java application run
+ -kill 
    + java application kill
+ -python
    + 자동계측 python application run
+ -kill-python
    + python application kill
+ -logs [서비스명]
    + docker compose logs -f 