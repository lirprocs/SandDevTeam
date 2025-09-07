# Реализован интерфейс Pool, покрыт тестами[![Test](https://github.com/lirprocs/SandDevTeam/actions/workflows/test.yaml/badge.svg)](https://github.com/lirprocs/SandDevTeam/actions/workflows/test.yaml)
```go
type Pool interface { 
// Submit - добавить задачу в пул. 
// Если пул не имеет свободных воркеров, то задачу нужно добавить в очередь. 
Submit(task func()) error 

// Stop - остановить воркер пул, дождаться выполнения всех добавленных ранее в очередь задач. 
Stop() error 
}
```


## Дополнительно: 
- Ограничен размер очереди. Если очередь переполнена то возвращается ошибка из метода Submit. 
- Реализована поддержка хука, который вызывается после выполнения каждой задачи.
