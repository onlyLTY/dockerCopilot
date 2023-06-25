import schedule
import time
import os


def job():
    os.system("python /app/manage.py check_update_command")


schedule.every(12).hours.do(job)

while True:
    schedule.run_pending()
    time.sleep(1)
