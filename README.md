# Команды docker для создания образа и запуска контейнера, запускаются из папаки проекта. Предполагается что docker уже установлен
# создание образа
sudo docker build -t pipeline .
# запуск контейнера в интерактивном режиме для взаимодействия с приложением
sudo docker run -i --rm --name=pipeline pipeline